// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package interop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cB-Sigma/evroaming/oscp"
)

// oscpImage is the OSCP peer's image, built from ./oscppeer.
const oscpImage = "sigma-evroaming-interop:pyoscp-0.1.0"

// startOSCPPeer runs pyoscp as a Capacity Provider with its port
// published on 127.0.0.1.
func startOSCPPeer(t *testing.T) *peer {
	t.Helper()
	out, err := exec.Command("docker", "build", "-q", "-t", oscpImage, "oscppeer").CombinedOutput()
	if err != nil {
		t.Fatalf("docker build: %v\n%s", err, out)
	}
	p := &peer{t: t, name: fmt.Sprintf("sigma-evroaming-oscp-%d", time.Now().UnixNano()), port: freePort(t)}
	cmd := exec.Command("docker", "run", "--rm", "--name", p.name, "--add-host", "host.docker.internal:host-gateway",
		"-p", fmt.Sprintf("127.0.0.1:%d:5000", p.port), "-e", fmt.Sprintf("HOST_URL=http://127.0.0.1:%d", p.port),
		"-e", "PEER_TOKEN_A=peer-token-a", oscpImage)
	r, w := io.Pipe()
	cmd.Stdout, cmd.Stderr = w, w
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			p.mu.Lock()
			p.log = append(p.log, sc.Text())
			p.mu.Unlock()
		}
	}()
	go func() { _ = cmd.Wait(); _ = w.Close() }()
	t.Cleanup(func() {
		_ = exec.Command("docker", "stop", "-t", "2", p.name).Run()
		if t.Failed() || os.Getenv(envVar+"_LOG") == "1" {
			t.Logf("peer output:\n%s", p.output())
		}
	})
	for i := 0; i < 200; i++ {
		resp, err := http.Get(p.cpBase() + "/heartbeat")
		if err == nil {
			resp.Body.Close()
			return p
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("the OSCP peer did not answer\n%s", p.output())
	return nil
}

func (p *peer) cpBase() string { return fmt.Sprintf("http://127.0.0.1:%d/oscp/cp/2.0", p.port) }

// fp is our Flexibility Provider side: it records and validates what the
// peer sends.
type fp struct {
	mu   sync.Mutex
	tok  string
	got  []string
	errs []string
	reg  oscp.Register
}

func (f *fp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ep := oscp.CanonicalEndpoint(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, ep)
	if r.Header.Get(oscp.HeaderAuthorization) != "Token "+f.tok {
		f.errs = append(f.errs, ep+": token "+r.Header.Get(oscp.HeaderAuthorization))
	}
	if r.Header.Get(oscp.HeaderRequestID) == "" {
		f.errs = append(f.errs, ep+": no X-Request-ID")
	}
	e, ok := oscp.EndpointByName(ep)
	if !ok {
		f.errs = append(f.errs, "unknown endpoint "+ep)
	} else if err := oscp.Validate(e.Schema, body); err != nil {
		f.errs = append(f.errs, fmt.Sprintf("%s: %v (%s)", ep, err, body))
	}
	if ep == oscp.EPRegister {
		_ = json.Unmarshal(body, &f.reg)
	}
	w.WriteHeader(http.StatusNoContent)
}

// The library as a Flexibility Provider against pyoscp 0.1.0 as a
// Capacity Provider: registration both ways (the peer's Register back is
// validated against the OCA schemas), then every message an FP sends a
// CP, accepted by the peer's models; a malformed message is refused.
func TestOSCPAgainstPyoscp(t *testing.T) {
	needDocker(t)
	p := startOSCPPeer(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fp{tok: oscp.NewToken()}
	srv := &http.Server{Handler: f, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()
	ourBase := fmt.Sprintf("http://host.docker.internal:%d/oscp/fp/2.0", ln.Addr().(*net.TCPAddr).Port)
	ctx := context.Background()
	hdr := func(tok string) http.Header {
		h := http.Header{}
		h.Set(oscp.HeaderAuthorization, "Token "+tok)
		h.Set(oscp.HeaderRequestID, oscp.NewRequestID())
		if !strings.HasPrefix(tok, "peer-token-a") { // a first Register carries no correlation ID
			h.Set(oscp.HeaderCorrelationID, oscp.NewRequestID())
		}
		return h
	}
	post := func(ep, tok string, v any) error {
		b, _ := json.Marshal(v)
		if e, ok := oscp.EndpointByName(ep); ok {
			if err := oscp.Validate(e.Schema, b); err != nil {
				return fmt.Errorf("our own message is invalid: %w", err)
			}
		}
		return oscp.Post(ctx, nil, http.MethodPost, p.cpBase()+"/"+ep, hdr(tok), b)
	}

	if err := post(oscp.EPRegister, "peer-token-a", oscp.Register{Token: f.tok, VersionURL: []oscp.VersionURL{{Version: oscp.Version, BaseURL: ourBase}}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	var tokC string
	for i := 0; i < 100; i++ {
		f.mu.Lock()
		tokC = f.reg.Token
		f.mu.Unlock()
		if tokC != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if tokC == "" {
		t.Fatal("the peer did not register back")
	}
	t.Logf("passed: Register both ways (the peer's Register is valid against the OCA schema)")
	iv := 30
	now := func(d time.Duration) string { return time.Now().Add(d).UTC().Format("2006-01-02T15:04:05.000Z07:00") }
	blocks := []oscp.ForecastedBlock{{Capacity: 20, Phase: "ALL", Unit: oscp.UnitKW, StartTime: now(time.Hour), EndTime: now(2 * time.Hour)}}
	for _, m := range []struct {
		ep, peerLine string
		v            any
	}{
		{oscp.EPHeartbeat, "", oscp.Heartbeat{OfflineModeAt: now(5 * time.Minute)}},
		{oscp.EPHandshakeAcknowledgment, "", oscp.Handshake{RequiredBehaviour: &oscp.RequiredBehaviour{HeartbeatInterval: &iv, MeasurementConfiguration: []string{oscp.MeasurementContinuous}}}},
		{oscp.EPAdjustGroupCapacityForecast, "PEER adjust group-1 1", oscp.GroupCapacityForecast{GroupID: "group-1", Type: oscp.TypeConsumption, ForecastedBlocks: blocks}},
		{oscp.EPGroupCapacityComplianceError, "PEER compliance-error", oscp.GroupCapacityComplianceError{Message: "cannot comply", ForecastedBlocks: blocks}},
		{oscp.EPUpdateGroupMeasurements, "PEER group-measurements group-1 1", oscp.UpdateGroupMeasurements{GroupID: "group-1",
			Measurements: []oscp.EnergyMeasurement{{Value: 12.5, Phase: "ALL", Unit: oscp.UnitKWH, Direction: "IMPORT", EnergyType: "TOTAL", MeasureTime: now(-time.Minute)}}}},
	} {
		if err := post(m.ep, tokC, m.v); err != nil {
			if m.ep == oscp.EPHeartbeat {
				// The peer parses offline_mode_at as "YYYY-MM-DD hh:mm:ss",
				// not as the RFC 3339 date-time of 5.x and the schema.
				t.Logf("peer deviation: heartbeat with an RFC 3339 offline_mode_at: %v", err)
				continue
			}
			t.Errorf("%s: %v", m.ep, err)
			continue
		}
		if m.peerLine != "" {
			handled := false
			for i := 0; i < 50 && !handled; i++ {
				handled = p.saw(strings.Fields(m.peerLine)[1:]...)
				time.Sleep(50 * time.Millisecond)
			}
			if !handled {
				t.Errorf("%s: the peer did not handle it", m.ep)
				continue
			}
		}
		t.Logf("passed: %s accepted", m.ep)
	}
	// A malformed message (a capacity that is not a number) is refused by
	// the peer's models.
	var se *oscp.SendError
	err = oscp.Post(ctx, nil, http.MethodPost, p.cpBase()+"/"+oscp.EPAdjustGroupCapacityForecast, hdr(tokC),
		[]byte(`{"group_id":"g","type":"CONSUMPTION","forecasted_blocks":[{"capacity":"lots","phase":"ALL","unit":"KW","start_time":"x","end_time":"y"}]}`))
	if !errors.As(err, &se) || se.Status != http.StatusBadRequest {
		t.Errorf("malformed message: %v", err)
	} else {
		t.Logf("passed: a malformed message was refused with 400")
	}
	if err := post(oscp.EPHeartbeat, "wrong-token", oscp.Heartbeat{OfflineModeAt: now(time.Minute)}); err == nil {
		t.Error("an unknown token was accepted")
	} else {
		t.Logf("passed: an unknown token was refused (%v)", err)
	}
	time.Sleep(6 * time.Second) // one round of the peer's background job
	f.mu.Lock()
	defer f.mu.Unlock()
	t.Logf("the peer sent us: %v", f.got)
	for _, e := range f.errs {
		if strings.HasPrefix(e, "heartbeat: /offline_mode_at") {
			t.Logf("peer deviation: its heartbeat's offline_mode_at is not an RFC 3339 date-time: %s", e)
			continue
		}
		t.Errorf("the peer's message: %s", e)
	}
}
