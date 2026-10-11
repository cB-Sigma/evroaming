package oscp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTP headers of 4.1.2.
const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderSegmentIndex  = "X-Segment-Index"
	HeaderSegmentCount  = "X-Segment-Count"
)

// NewRequestID returns a unique X-Request-ID (a random UUID).
func NewRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// NewToken returns a random token of 43 characters (32 bytes, base64url),
// like the example of 4.1.2.
func NewToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// SendError is a refused request: the peer's HTTP status and message.
type SendError struct {
	Status  int
	Message string
}

func (e *SendError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("the peer answered %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("the peer answered %d", e.Status)
}

// Segments splits body into segments of at most max bytes (4.1.2.1); a
// max of 0 or a body that fits gives one segment.
func Segments(body []byte, max int) [][]byte {
	if max <= 0 || len(body) <= max {
		return [][]byte{body}
	}
	var segs [][]byte
	for i := 0; i < len(body); i += max {
		segs = append(segs, body[i:min(i+max, len(body))])
	}
	return segs
}

// Post sends one request (a message or one segment of it) with the
// headers hdr. A 2xx answer is a success (204 is the answer of 4.1.3.1);
// anything else is a *SendError with the error body's message.
func Post(ctx context.Context, client *http.Client, method, target string, hdr http.Header, body []byte) error {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rd)
	if err != nil {
		return err
	}
	req.Header = hdr
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode/100 == 2 { // other 2xx are taken as success too
		return nil
	}
	var eb ErrorBody
	_ = json.Unmarshal(b, &eb)
	return &SendError{Status: resp.StatusCode, Message: eb.Message}
}
