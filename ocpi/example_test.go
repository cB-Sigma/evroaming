// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/cB-Sigma/evroaming/ocpi"
)

// A partner's versions endpoint, read with its registration token: the
// first step of the credentials handshake.
func Example() {
	partner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := ocpi.TokenCandidates(r.Header.Get("Authorization")); !ok || c[0].Token != "token-A" {
			ocpi.WriteReply(w, http.StatusUnauthorized, ocpi.StatusClientError, "unknown token", nil, time.Now())
			return
		}
		ocpi.WriteReply(w, http.StatusOK, ocpi.StatusOK, "", []ocpi.VersionInfo{{Version: "2.2.1", URL: "https://partner.example/ocpi/2.2.1"}}, time.Now())
	}))
	defer partner.Close()

	c := &ocpi.Client{From: ocpi.Party{CountryCode: "NL", PartyID: "CPO"}}
	rep, err := c.Send(context.Background(), ocpi.Request{Method: http.MethodGet, URL: partner.URL + "/versions"}, "token-A", false)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rep.Envelope.StatusCode, string(rep.Envelope.Data))
	// Output: 1000 [{"version":"2.2.1","url":"https://partner.example/ocpi/2.2.1"}]
}
