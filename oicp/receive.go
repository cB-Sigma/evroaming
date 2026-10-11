// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package oicp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Paths of the operations Hubject sends to a CPO, as the CPO OpenAPI
// description lists them below the CPO's base URL; pv is the EMP's
// ProviderID. A CPO may also give Hubject other URLs; Handler matches
// these paths by their endings, so either works.

// PathRemoteStart is eRoamingAuthorizeRemoteStart.
func PathRemoteStart(pv string) string {
	return "/charging/v21/providers/" + url.PathEscape(pv) + "/authorize-remote/start"
}

// PathRemoteStop is eRoamingAuthorizeRemoteStop.
func PathRemoteStop(pv string) string {
	return "/charging/v21/providers/" + url.PathEscape(pv) + "/authorize-remote/stop"
}

// PathReservationStart is eRoamingAuthorizeRemoteReservationStart.
func PathReservationStart(pv string) string {
	return "/reservation/v11/providers/" + url.PathEscape(pv) + "/reservation-start-request"
}

// PathReservationStop is eRoamingAuthorizeRemoteReservationStop.
func PathReservationStop(pv string) string {
	return "/reservation/v11/providers/" + url.PathEscape(pv) + "/reservation-stop-request"
}

// Handler serves the operations Hubject sends to a CPO: remote start and
// stop, and remote reservation start and stop. Each callback returns the
// acknowledgement to answer with; a nil callback answers Result false
// with status code 320 (service not available). The request's
// SessionID is copied into the acknowledgement when the callback leaves
// it empty. A body that is not the operation's JSON object is answered
// with status code 022 (data error). Mutual TLS is the server's: run
// the handler behind a tls.Config that requires Hubject's client
// certificate.
type Handler struct {
	RemoteStart      func(ctx context.Context, m RemoteStart) Acknowledgement
	RemoteStop       func(ctx context.Context, m RemoteStop) Acknowledgement
	ReservationStart func(ctx context.Context, m RemoteReservationStart) Acknowledgement
	ReservationStop  func(ctx context.Context, m RemoteReservationStop) Acknowledgement
	// MaxBodyBytes caps the request read (0: 1 MiB).
	MaxBodyBytes int64
}

// Operation names the Hubject-to-CPO operation a path ends with: one of
// "RemoteStart", "RemoteStop", "ReservationStart", "ReservationStop", or
// "" for none.
func Operation(path string) string {
	p := strings.TrimRight(path, "/")
	switch {
	case strings.HasSuffix(p, "/authorize-remote/start"):
		return "RemoteStart"
	case strings.HasSuffix(p, "/authorize-remote/stop"):
		return "RemoteStop"
	case strings.HasSuffix(p, "/reservation-start-request"):
		return "ReservationStart"
	case strings.HasSuffix(p, "/reservation-stop-request"):
		return "ReservationStop"
	}
	return ""
}

// ServeHTTP answers a POST of one of the four operations; other methods
// get 405 and other paths 404.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	op := Operation(r.URL.Path)
	if op == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	max := h.MaxBodyBytes
	if max <= 0 {
		max = 1 << 20
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, max))
	if err != nil {
		WriteAck(w, NewAck(false, CodeDataError, "the request could not be read", ""))
		return
	}
	ctx := r.Context()
	var ack Acknowledgement
	switch op {
	case "RemoteStart":
		var m RemoteStart
		if bad(w, body, &m) {
			return
		}
		ack = call(ctx, h.RemoteStart, m, m.SessionID)
	case "RemoteStop":
		var m RemoteStop
		if bad(w, body, &m) {
			return
		}
		ack = call(ctx, h.RemoteStop, m, m.SessionID)
	case "ReservationStart":
		var m RemoteReservationStart
		if bad(w, body, &m) {
			return
		}
		ack = call(ctx, h.ReservationStart, m, m.SessionID)
	case "ReservationStop":
		var m RemoteReservationStop
		if bad(w, body, &m) {
			return
		}
		ack = call(ctx, h.ReservationStop, m, m.SessionID)
	}
	WriteAck(w, ack)
}

func bad(w http.ResponseWriter, body []byte, v any) bool {
	if err := json.Unmarshal(body, v); err != nil {
		WriteAck(w, NewAck(false, CodeDataError, "the body is not the operation's JSON object", ""))
		return true
	}
	return false
}

func call[M any](ctx context.Context, f func(context.Context, M) Acknowledgement, m M, session string) Acknowledgement {
	if f == nil {
		return NewAck(false, CodeServiceUnavailable, "this operation is not offered", session)
	}
	a := f(ctx, m)
	if a.SessionID == "" {
		a.SessionID = session
	}
	return a
}

// NewAck is an acknowledgement with a result, a status code, its
// description and a SessionID.
func NewAck(result bool, code, description, session string) Acknowledgement {
	return Acknowledgement{Result: result, StatusCode: StatusCode{Code: code, Description: description}, SessionID: session}
}

// WriteAck writes an acknowledgement as a 200 answer.
func WriteAck(w http.ResponseWriter, a Acknowledgement) {
	b, _ := json.Marshal(a)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}
