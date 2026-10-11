package ocpi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SetRequestIDs copies X-Request-ID and X-Correlation-ID of a request to
// the answer, making new ones where the request has none (4.1.5).
func SetRequestIDs(w http.ResponseWriter, r *http.Request) {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		w.Header().Set("X-Request-ID", id)
	} else {
		w.Header().Set("X-Request-ID", NewID())
	}
	if id := r.Header.Get("X-Correlation-ID"); id != "" {
		w.Header().Set("X-Correlation-ID", id)
	} else {
		w.Header().Set("X-Correlation-ID", NewID())
	}
}

// WriteReply writes the response envelope (4.1.7) with status code code,
// message msg, data (nil for none) and the timestamp now.
func WriteReply(w http.ResponseWriter, httpStatus, code int, msg string, data any, now time.Time) {
	env := Envelope{StatusCode: code, StatusMessage: msg, Timestamp: FormatTime(now)}
	if data != nil {
		b, err := json.Marshal(data)
		if err == nil {
			env.Data = b
		}
	}
	b, _ := json.Marshal(env)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(b)
}

// SplitPath splits a URL path into its unescaped segments.
func SplitPath(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s == "" {
			continue
		}
		if u, err := url.PathUnescape(s); err == nil {
			s = u
		}
		out = append(out, s)
	}
	return out
}

// TokenCandidate is a reading of an Authorization header's credentials
// token.
type TokenCandidate struct {
	Token string
	// Raw is set when the token is taken as it is, not Base64-decoded
	// (OCPI before 2.2).
	Raw bool
}

// TokenCandidates reads "Token <base64>" (4.1.2): the decoded token when
// the value is Base64 of printable ASCII, then the value as it is. ok is
// false when the header has no token.
func TokenCandidates(authorization string) (cands []TokenCandidate, ok bool) {
	h := strings.TrimSpace(authorization)
	v, ok := cutFold(h, "Token ")
	if !ok || strings.TrimSpace(v) == "" {
		return nil, false
	}
	v = strings.TrimSpace(v)
	if b, err := base64.StdEncoding.DecodeString(v); err == nil && PrintableASCII(string(b)) {
		cands = append(cands, TokenCandidate{string(b), false})
	} else if b, err := base64.RawStdEncoding.DecodeString(v); err == nil && PrintableASCII(string(b)) {
		cands = append(cands, TokenCandidate{string(b), false})
	}
	return append(cands, TokenCandidate{v, true}), true
}

func cutFold(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return "", false
}

// SetRoutingHeaders sets the routing headers of an answer: the request's
// from becomes our to and the other way round (4.1.8); self is our party
// when the request names no to.
func SetRoutingHeaders(w http.ResponseWriter, r *http.Request, self Party) {
	h := w.Header()
	if cc, pid := r.Header.Get("OCPI-from-country-code"), r.Header.Get("OCPI-from-party-id"); cc != "" {
		h.Set("OCPI-to-country-code", cc)
		h.Set("OCPI-to-party-id", pid)
	}
	cc, pid := r.Header.Get("OCPI-to-country-code"), r.Header.Get("OCPI-to-party-id")
	if cc == "" {
		cc, pid = self.CountryCode, self.PartyID
	}
	h.Set("OCPI-from-country-code", cc)
	h.Set("OCPI-from-party-id", pid)
}

// FromHeaders is the requesting party of the OCPI-from headers (ok false
// when they are missing).
func FromHeaders(r *http.Request) (Party, bool) {
	cc, pid := r.Header.Get("OCPI-from-country-code"), r.Header.Get("OCPI-from-party-id")
	if cc != "" && pid != "" {
		return Party{CountryCode: Upper(cc), PartyID: Upper(pid)}, true
	}
	return Party{}, false
}

// Item is one object of a list we serve, with its last_updated.
type Item struct {
	Updated time.Time
	Value   any
}

// Paginate selects the page of items a list request asks for (4.1.4.1):
// date_from (inclusive) and date_to (exclusive) on last_updated, offset
// and limit (at most pageLimit), oldest first. It sets X-Total-Count,
// X-Limit and, when more follow, a Link to the next page at self (the
// public URL of the list) that keeps the filters. A bad filter is an
// error to answer with StatusInvalidParams.
func Paginate(h http.Header, r *http.Request, items []Item, pageLimit int, self string) ([]any, error) {
	q := r.URL.Query()
	var from, to time.Time
	if s := q.Get("date_from"); s != "" {
		if t, ok := ParseTime(s); ok {
			from = t
		} else {
			return nil, fmt.Errorf("date_from is not a DateTime")
		}
	}
	if s := q.Get("date_to"); s != "" {
		if t, ok := ParseTime(s); ok {
			to = t
		} else {
			return nil, fmt.Errorf("date_to is not a DateTime")
		}
	}
	var sel []Item
	for _, it := range items {
		if !from.IsZero() && it.Updated.Before(from) {
			continue
		}
		if !to.IsZero() && !it.Updated.Before(to) {
			continue
		}
		sel = append(sel, it)
	}
	sort.SliceStable(sel, func(i, j int) bool { return sel[i].Updated.Before(sel[j].Updated) })
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	limit := pageLimit
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l < limit {
		limit = l
	}
	h.Set("X-Total-Count", strconv.Itoa(len(sel)))
	h.Set("X-Limit", strconv.Itoa(pageLimit))
	end := offset + limit
	if end > len(sel) {
		end = len(sel)
	}
	out := []any{}
	if offset < len(sel) {
		for _, it := range sel[offset:end] {
			out = append(out, it.Value)
		}
	}
	if end < len(sel) {
		nq := url.Values{}
		for k, v := range q {
			nq[k] = v
		}
		nq.Set("offset", strconv.Itoa(end))
		nq.Set("limit", strconv.Itoa(limit))
		h.Set("Link", fmt.Sprintf("<%s?%s>; rel=\"next\"", self, nq.Encode()))
	}
	return out, nil
}

// Decode reads a request body; unknown members are ignored.
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("the body is not the expected JSON object: %w", err)
	}
	return nil
}

// RemoteIsLocal says whether the request came from this machine.
func RemoteIsLocal(r *http.Request) bool {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// Upper is s trimmed and in upper case (country codes and party IDs).
func Upper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// PrintableASCII says whether s is non-empty printable ASCII without
// spaces.
func PrintableASCII(s string) bool {
	for _, r := range s {
		if r <= ' ' || r > '~' {
			return false
		}
	}
	return s != ""
}
