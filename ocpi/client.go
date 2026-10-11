package ocpi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// CallError is a partner's refusal: an HTTP error or an OCPI status code
// outside 1xxx.
type CallError struct {
	HTTP    int
	Code    int
	Message string
}

func (e *CallError) Error() string {
	switch {
	case e.Code != 0 && e.HTTP != 0 && e.HTTP != http.StatusOK:
		return fmt.Sprintf("HTTP %d, OCPI status %d: %s", e.HTTP, e.Code, e.Message)
	case e.Code != 0:
		return fmt.Sprintf("OCPI status %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", e.HTTP, e.Message)
}

// NotFound says whether err is a 404 or an OCPI "unknown" status.
func NotFound(err error) bool {
	var ce *CallError
	return errors.As(err, &ce) && (ce.HTTP == http.StatusNotFound || ce.Code == StatusUnknownToken || ce.Code == StatusUnknownLocation)
}

// Request is one call to a partner.
type Request struct {
	Method string
	URL    string
	Body   any
	// Token overrides the partner's token (token A during registration).
	// Client.Send ignores it; a Doer uses it to choose the token.
	Token string
	// To is the party addressed (functional modules); nil for
	// configuration modules, which carry no routing headers (4.1.8).
	To          *Party
	Correlation string
}

// Reply is a partner's answer.
type Reply struct {
	Envelope Envelope
	Header   http.Header
}

// Doer sends a request to one partner with that partner's credentials. An
// application implements it around Client.Send, with its own token store
// and retry policy.
type Doer func(ctx context.Context, r Request) (*Reply, error)

// NewID returns a random UUID for X-Request-ID and X-Correlation-ID.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Client sends OCPI requests.
type Client struct {
	// HTTP is the HTTP client (nil: http.DefaultClient).
	HTTP *http.Client
	// From is our party, sent in the OCPI-from headers of functional
	// module calls.
	From Party
	// MaxBodyBytes caps the answer read (0: 4 MiB).
	MaxBodyBytes int
}

// Send sends r with the credentials token tok: Base64-encoded as 2.2.1
// requires (4.1.2), or as it is when raw is set (earlier versions). An
// HTTP status outside 2xx or an OCPI status outside 1xxx is a *CallError;
// an empty 2xx answer is a success.
func (c *Client) Send(ctx context.Context, r Request, tok string, raw bool) (*Reply, error) {
	var body io.Reader
	if r.Body != nil {
		b, err := json.Marshal(r.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, body)
	if err != nil {
		return nil, err
	}
	if raw {
		req.Header.Set("Authorization", "Token "+tok)
	} else {
		req.Header.Set("Authorization", "Token "+base64.StdEncoding.EncodeToString([]byte(tok)))
	}
	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", NewID())
	corr := r.Correlation
	if corr == "" {
		corr = NewID()
	}
	req.Header.Set("X-Correlation-ID", corr)
	if r.To != nil {
		req.Header.Set("OCPI-to-country-code", r.To.CountryCode)
		req.Header.Set("OCPI-to-party-id", r.To.PartyID)
		req.Header.Set("OCPI-from-country-code", c.From.CountryCode)
		req.Header.Set("OCPI-from-party-id", c.From.PartyID)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	max := c.MaxBodyBytes
	if max <= 0 {
		max = 4 << 20
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, int64(max)))
	rep := &Reply{Header: resp.Header}
	jerr := json.Unmarshal(b, &rep.Envelope)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := rep.Envelope.StatusMessage
		if msg == "" {
			msg = strings.TrimSpace(string(b))
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		return rep, &CallError{HTTP: resp.StatusCode, Code: rep.Envelope.StatusCode, Message: msg}
	}
	if jerr != nil {
		if len(bytes.TrimSpace(b)) == 0 {
			return rep, nil // an empty 2xx: taken as success
		}
		return rep, &CallError{HTTP: resp.StatusCode, Message: "the answer is not an OCPI response: " + jerr.Error()}
	}
	if rep.Envelope.StatusCode != 0 && (rep.Envelope.StatusCode < 1000 || rep.Envelope.StatusCode > 1999) {
		return rep, &CallError{HTTP: resp.StatusCode, Code: rep.Envelope.StatusCode, Message: rep.Envelope.StatusMessage}
	}
	return rep, nil
}

var linkNext = regexp.MustCompile(`<([^>]+)>\s*;\s*rel="?next"?`)

// GetAll reads every page of a list (4.1.4.1): it follows the Link
// header's next page until there is none, at most 1000 pages. total is
// the last X-Total-Count seen (-1 when none was).
func GetAll(ctx context.Context, do Doer, u string, to *Party, query url.Values) (items []json.RawMessage, total int, err error) {
	const maxPages = 1000
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + query.Encode()
	}
	var out []json.RawMessage
	total = -1
	for page := 0; u != "" && page < maxPages; page++ {
		rep, err := do(ctx, Request{Method: http.MethodGet, URL: u, To: to})
		if err != nil {
			return out, total, err
		}
		var list []json.RawMessage
		if len(rep.Envelope.Data) > 0 && string(rep.Envelope.Data) != "null" {
			if err := json.Unmarshal(rep.Envelope.Data, &list); err != nil {
				return out, total, fmt.Errorf("page %d is not a list: %w", page+1, err)
			}
		}
		out = append(out, list...)
		if n, err := strconv.Atoi(rep.Header.Get("X-Total-Count")); err == nil {
			total = n
		}
		u = ""
		for _, l := range rep.Header.Values("Link") {
			if m := linkNext.FindStringSubmatch(l); m != nil {
				u = m[1]
			}
		}
		if len(list) == 0 {
			break
		}
	}
	return out, total, nil
}

// JoinURL appends path segments to an endpoint URL.
func JoinURL(base string, seg ...string) string {
	u := strings.TrimRight(base, "/")
	for _, s := range seg {
		u += "/" + url.PathEscape(s)
	}
	return u
}
