// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A list served with Paginate is read back whole by GetAll through the
// Link headers, oldest first, with date_from applied (4.1.4.1).
func TestPaginateGetAll(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var items []Item
	for i := 0; i < 7; i++ {
		items = append(items, Item{Updated: base.Add(time.Duration(7-i) * time.Minute), Value: map[string]any{"id": fmt.Sprint(7 - i)}})
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, err := Paginate(w.Header(), r, items, 3, srv.URL+r.URL.Path)
		if err != nil {
			WriteReply(w, http.StatusBadRequest, StatusInvalidParams, err.Error(), nil, time.Now())
			return
		}
		WriteReply(w, http.StatusOK, StatusOK, "", out, time.Now())
	}))
	defer srv.Close()
	c := &Client{}
	do := func(ctx context.Context, r Request) (*Reply, error) { return c.Send(ctx, r, "tok", false) }
	got, total, err := GetAll(context.Background(), do, srv.URL+"/locations", nil, map[string][]string{"date_from": {FormatTime(base.Add(2 * time.Minute))}})
	if err != nil || total != 6 || len(got) != 6 {
		t.Fatalf("%d of %d: %v", len(got), total, err)
	}
	var first map[string]any
	_ = json.Unmarshal(got[0], &first)
	if first["id"] != "2" {
		t.Errorf("first %v", first)
	}
	_, _, err = GetAll(context.Background(), do, srv.URL+"/locations", nil, map[string][]string{"date_from": {"yesterday"}})
	var ce *CallError
	if !errors.As(err, &ce) || ce.Code != StatusInvalidParams {
		t.Errorf("bad date_from: %v", err)
	}
}

// The credentials token is read as Base64 first, then as it is (4.1.2).
func TestTokenCandidates(t *testing.T) {
	c, ok := TokenCandidates("token " + base64.StdEncoding.EncodeToString([]byte("abc-123")))
	if !ok || len(c) != 2 || c[0] != (TokenCandidate{"abc-123", false}) || !c[1].Raw {
		t.Errorf("%v %v", c, ok)
	}
	if _, ok := TokenCandidates("Bearer x"); ok {
		t.Error("Bearer accepted")
	}
}

// FetchEndpoints takes 2.2 when 2.2.1 is not offered, and refuses a
// partner with neither.
func TestFetchEndpoints(t *testing.T) {
	var devs []string
	do := func(_ context.Context, r Request) (*Reply, error) {
		switch r.URL {
		case "v":
			return &Reply{Envelope: Envelope{Data: json.RawMessage(`[{"version":"2.1.1","url":"x"},{"version":"2.2","url":"d"}]`)}}, nil
		case "d":
			return &Reply{Envelope: Envelope{Data: json.RawMessage(`{"version":"2.2","endpoints":[{"identifier":"credentials","role":"RECEIVER","url":"c"}]}`)}}, nil
		case "old":
			return &Reply{Envelope: Envelope{Data: json.RawMessage(`{"version":"2.1.1","url":"x"}`)}}, nil
		}
		return nil, errors.New("unexpected " + r.URL)
	}
	ver, eps, err := FetchEndpoints(context.Background(), do, "v", "a", func(code, _ string) { devs = append(devs, code) })
	if err != nil || ver != "2.2" || EndpointURL(eps, ModCredentials) != "c" || len(devs) != 1 || devs[0] != "ocpi.versionVariant" {
		t.Errorf("%s %v %v %v", ver, eps, err, devs)
	}
	_, _, err = FetchEndpoints(context.Background(), do, "old", "a", nil)
	var ce *CallError
	if !errors.As(err, &ce) || ce.Code != StatusUnsupportedVer {
		t.Errorf("no 2.2.x: %v", err)
	}
}
