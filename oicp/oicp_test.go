// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package oicp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A Result false acknowledgement is a HubjectError; system codes and 5xx
// are worth sending again, data errors are not.
func TestPostAck(t *testing.T) {
	code := CodeHubjectSystem
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Result":false,"StatusCode":{"Code":"` + code + `","Description":"busy"}}`))
	}))
	defer srv.Close()
	var link []bool
	c := &Client{BaseURL: srv.URL, Link: func(ok bool, _ string) { link = append(link, ok) }}
	_, err := c.PostAck(context.Background(), PathEvseStatus("DE*ABC"), PushEvseStatus{})
	var he *HubjectError
	if !errors.As(err, &he) || he.Code != CodeHubjectSystem || !Retryable(err) {
		t.Errorf("%v", err)
	}
	code = CodeDataError
	if _, err := c.PostAck(context.Background(), PathEvseStatus("DE*ABC"), PushEvseStatus{}); Retryable(err) {
		t.Errorf("data error retryable: %v", err)
	}
	if len(link) != 4 || !link[0] || link[1] {
		t.Errorf("link reports %v", link)
	}
}

// Identifier forms (OperatorIDType, EvseIDType, ProviderIDType) and their
// comparison with and without separators.
func TestIdentifiers(t *testing.T) {
	if !ValidOperatorID("DE*ABC") || !ValidEvseID("DE*ABC*E123") || !ValidProviderID("DE-XYZ") || ValidEvseID("DE*ABC") {
		t.Error("identifier forms")
	}
	if NormID("de*abc*e1") != NormID("DEABCE1") || !ValidHotline("+4930123456") || ValidHotline("030123") {
		t.Error("NormID or hotline")
	}
}
