// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fake is one value implementing two module interfaces, to check that
// the Router dispatches by route, not by the value's type.
type fake struct{ patched map[string]any }

func (f *fake) Locations(context.Context, *Call) ([]Item, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []Item{{now, Location{ID: "L1"}}, {now.Add(time.Minute), Location{ID: "L2"}}, {now.Add(2 * time.Minute), Location{ID: "L3"}}}, nil
}

func (f *fake) Location(_ context.Context, _ *Call, ids ...string) (any, error) {
	if ids[0] == "L1" {
		return Location{ID: "L1"}, nil
	}
	return nil, ErrNotFound
}

func (f *fake) Sessions(context.Context, *Call) ([]Item, error) {
	return []Item{{time.Now(), Session{ID: "S1"}}}, nil
}

func (f *fake) SetChargingPreferences(context.Context, *Call, string, ChargingPreferences) (string, error) {
	return PrefsAccepted, nil
}

func (f *fake) GetToken(context.Context, *Call, Party, string, string) (Token, error) {
	return Token{}, ErrNotFound
}
func (f *fake) PutToken(context.Context, *Call, Party, Token) error { return nil }
func (f *fake) PatchToken(_ context.Context, _ *Call, _ Party, _, _ string, fields map[string]any) error {
	f.patched = fields
	return nil
}

func TestRouter(t *testing.T) {
	f := &fake{}
	var srv *httptest.Server
	rt := &Router{Self: Party{"NL", "CPO"}, PageLimit: 2, LocationsSender: f, SessionsSender: f, TokensReceiver: f,
		Authenticate: func(_ *http.Request, tok string) (any, bool, bool) {
			return tok, tok == "token-a", tok == "token-c" || tok == "token-a"
		}}
	srv = httptest.NewServer(rt)
	defer srv.Close()
	rt.BaseURL = srv.URL + "/ocpi"
	c := &Client{From: Party{"DE", "MSP"}}
	as := func(tok string) Doer {
		return func(ctx context.Context, r Request) (*Reply, error) { return c.Send(ctx, r, tok, false) }
	}
	ctx := context.Background()
	ver, eps, err := FetchEndpoints(ctx, as("token-a"), rt.VersionsURL(), "token-a", nil)
	if err != nil || ver != Version || len(eps) != 3 {
		t.Fatalf("%s %v %v", ver, eps, err)
	}
	locs, total, err := List[Location](ctx, as("token-c"), EndpointURL(eps, ModLocations), &Party{"NL", "CPO"}, nil)
	if err != nil || total != 3 || len(locs) != 3 || locs[2].ID != "L3" {
		t.Errorf("locations %v %d %v", locs, total, err)
	}
	if ss, _, err := GetSessions(ctx, as("token-c"), EndpointURL(eps, ModSessions), nil, nil); err != nil || len(ss) != 1 || ss[0].ID != "S1" {
		t.Errorf("sessions %v %v", ss, err)
	}
	var l Location
	err = GetLocation(ctx, as("token-c"), EndpointURL(eps, ModLocations), nil, &l, "L7")
	var ce *CallError
	if !errors.As(err, &ce) || ce.HTTP != http.StatusNotFound || ce.Code != StatusUnknownLocation {
		t.Errorf("unknown location: %v", err)
	}
	if _, _, err := GetLocations(ctx, as("token-a"), EndpointURL(eps, ModLocations), nil, nil); !errors.As(err, &ce) || ce.HTTP != http.StatusUnauthorized {
		t.Errorf("registration token on a module: %v", err)
	}
	if _, _, err := GetLocations(ctx, as("nobody"), EndpointURL(eps, ModLocations), nil, nil); !errors.As(err, &ce) || ce.HTTP != http.StatusUnauthorized {
		t.Errorf("unknown token: %v", err)
	}
	tr := EndpointURL(eps, ModTokens)
	if err := PatchToken(ctx, as("token-c"), tr, nil, Party{"DE", "MSP"}, "T1", "", map[string]any{"valid": false, "last_updated": "2026-01-01T00:00:00Z"}); err != nil || f.patched["valid"] != false {
		t.Errorf("patch %v %v", f.patched, err)
	}
	_, err = as("token-c")(ctx, Request{Method: http.MethodPatch, URL: ObjectURL(tr, Party{"DE", "MSP"}, "T1"), Body: map[string]any{"valid": true}})
	if !errors.As(err, &ce) || ce.Code != StatusInvalidParams {
		t.Errorf("patch without last_updated: %v", err)
	}
	if _, err := Get[Token](ctx, as("token-c"), TokenURL(tr, Party{"DE", "MSP"}, "T9", TokenRFID), nil); !NotFound(err) {
		t.Errorf("unknown token object: %v", err)
	}
	_, err = as("token-c")(ctx, Request{Method: http.MethodDelete, URL: EndpointURL(eps, ModLocations)})
	if !errors.As(err, &ce) || ce.HTTP != http.StatusMethodNotAllowed {
		t.Errorf("DELETE on a sender: %v", err)
	}
}
