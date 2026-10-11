// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cB-Sigma/evroaming/ocpi"
)

// ours is the library's side: an ocpi.Router for the peer's callbacks
// (versions, version details, command and charging profile results).
type ours struct {
	mu      sync.Mutex
	results map[string]string
	tokenB  string
}

func (o *ours) CommandResult(_ context.Context, _ *ocpi.Call, path []string, r ocpi.CommandResult) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.results["commands/"+strings.Join(path, "/")] = r.Result
	return nil
}

func (o *ours) ProfileResult(_ context.Context, _ *ocpi.Call, path []string, raw json.RawMessage) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.results["profiles/"+strings.Join(path, "/")] = string(raw)
	return nil
}

func (o *ours) PutActiveProfile(context.Context, *ocpi.Call, string, ocpi.ActiveChargingProfile) error {
	return nil
}

func (o *ours) result(k string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.results[k]
}

func epFor(eps []ocpi.Endpoint, module, role string) string {
	for _, e := range eps {
		if e.Identifier == module && e.Role == role {
			return strings.TrimRight(e.URL, "/")
		}
	}
	return ""
}

const ts = "2026-10-11T08:00:00Z"

func location(id string) ocpi.Location {
	return ocpi.Location{CountryCode: "NL", PartyID: "SIG", ID: id, Publish: true, Name: "Demo " + id, Address: "Demo Street 1", City: "Utrecht",
		PostalCode: "3511", Country: "NLD", Coordinates: ocpi.GeoLocation{Latitude: "52.090736", Longitude: "5.121420"},
		ParkingType: ocpi.ParkingOnStreet, TimeZone: "Europe/Amsterdam", LastUpdated: ts,
		OpeningTimes: &ocpi.Hours{RegularHours: []ocpi.RegularHours{{Weekday: 1, PeriodBegin: "08:00", PeriodEnd: "20:00"}}},
		EnergyMix: &ocpi.EnergyMix{IsGreenEnergy: true, EnergySources: []ocpi.EnergySource{{Source: ocpi.SourceSolar, Percentage: 100}},
			SupplierName: "Demo Energy", EnergyProductName: "Green"},
		EVSEs: []ocpi.EVSE{{UID: id + "-E1", EVSEID: "NL*SIG*E" + id, Status: ocpi.EVSEAvailable, LastUpdated: ts,
			Capabilities: []string{ocpi.CapRemoteStartStopCapable},
			Connectors: []ocpi.Connector{{ID: "1", Standard: "IEC_62196_T2", Format: ocpi.FormatSocket, PowerType: ocpi.PowerAC3Phase,
				MaxVoltage: 230, MaxAmperage: 32, LastUpdated: ts}}}}}
}

// The library against extrawest_ocpi 2025.7.16 serving both the CPO and
// the eMSP role: the credentials handshake both ways round (the peer
// reads our versions and details), every module operation from both
// sides, and the results the peer POSTs to our response URLs.
func TestOCPIAgainstExtrawest(t *testing.T) {
	needDocker(t)
	p := startPeer(t, "peer-token-a")

	o := &ours{results: map[string]string{}, tokenB: "token-b-of-sigma"}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	rt := &ocpi.Router{BaseURL: fmt.Sprintf("http://host.docker.internal:%d/ocpi", port), Self: ocpi.Party{CountryCode: "NL", PartyID: "SIG"},
		CommandsSender: o, ChargingProfilesSender: o,
		Authenticate: func(_ *http.Request, tok string) (any, bool, bool) { return "peer", false, tok == o.tokenB }}
	srv := &http.Server{Handler: rt, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	host := fmt.Sprintf("127.0.0.1:%d", p.port)
	c := &ocpi.Client{From: ocpi.Party{CountryCode: "NL", PartyID: "SIG"},
		HTTP: &http.Client{Timeout: 20 * time.Second, Transport: httpsToHTTP{host: host, rt: http.DefaultTransport}}}
	var tokenC string
	do := func(ctx context.Context, r ocpi.Request) (*ocpi.Reply, error) {
		tok := r.Token
		if tok == "" {
			tok = tokenC
		}
		return c.Send(ctx, r, tok, false)
	}
	ctx := context.Background()
	var devs []string
	ver, eps, err := ocpi.FetchEndpoints(ctx, do, p.url()+"/versions", "peer-token-a", func(code, msg string) { devs = append(devs, msg) })
	if err != nil {
		t.Fatalf("versions and details: %v", err)
	}
	t.Logf("version %s, endpoints %v, deviations %q", ver, eps, devs)

	theirs, err := ocpi.RegisterCredentials(ctx, do, strings.TrimRight(ocpi.EndpointURL(eps, ocpi.ModCredentials), "/")+"/", "peer-token-a", ocpi.Credentials{
		Token: o.tokenB, URL: rt.VersionsURL(), Roles: []ocpi.CredentialsRole{
			{Role: ocpi.RoleCPO, PartyID: "SIG", CountryCode: "NL", BusinessDetails: ocpi.BusinessDetails{Name: "Sigma CPO"}},
			{Role: ocpi.RoleEMSP, PartyID: "SIG", CountryCode: "NL", BusinessDetails: ocpi.BusinessDetails{Name: "Sigma eMSP"}}}})
	if err != nil {
		t.Fatalf("registration: %v", err)
	}
	tokenC = theirs.Token
	if !p.saw("registered", "host.docker.internal") {
		t.Error("the peer did not read our versions and details")
	}
	own := ocpi.Party{CountryCode: "NL", PartyID: "SIG"}
	step := func(name string, f func() error) {
		t.Helper()
		if err := f(); err != nil {
			t.Errorf("%s: %v", name, err)
		} else {
			t.Logf("passed: %s", name)
		}
	}
	// peerDefect runs a step that fails because of the peer, as recorded
	// in the README; it is logged, and a pass is logged too.
	peerDefect := func(name, why string, f func() error) {
		t.Helper()
		if err := f(); err != nil {
			t.Logf("peer deviation: %s: %s (%v)", name, why, err)
		} else {
			t.Logf("passed (the recorded peer deviation did not show): %s", name)
		}
	}
	slash := func(u string) string { return u + "/" }

	// As a CPO, to the peer's eMSP Receiver interfaces.
	locR := epFor(eps, ocpi.ModLocations, ocpi.Receiver)
	for _, id := range []string{"L1", "L2", "L3"} {
		step("PUT location "+id, func() error { return ocpi.PutLocation(ctx, do, locR, nil, own, location(id), id) })
	}
	step("PUT EVSE", func() error { return ocpi.PutLocation(ctx, do, locR, nil, own, location("L1").EVSEs[0], "L1", "L1-E1") })
	step("PATCH location", func() error {
		return ocpi.PatchLocation(ctx, do, locR, nil, own, map[string]any{"name": "Patched", "last_updated": ts}, "L1")
	})
	step("GET location (receiver)", func() error {
		var l ocpi.Location
		if err := ocpi.GetLocation(ctx, do, locR, nil, &l, "NL", "SIG", "L1"); err != nil {
			return err
		}
		if l.Name != "Patched" {
			return fmt.Errorf("name %q", l.Name)
		}
		return nil
	})
	sess := ocpi.Session{CountryCode: "NL", PartyID: "SIG", ID: "S1", StartDateTime: ts, KWh: 1.5, AuthMethod: ocpi.AuthCommand,
		CdrToken:   ocpi.CdrToken{CountryCode: "BE", PartyID: "PEM", UID: "T1", Type: ocpi.TokenRFID, ContractID: "BE-PEM-C1"},
		LocationID: "L1", EVSEUID: "L1-E1", ConnectorID: "1", Currency: "EUR", Status: ocpi.SessionActive, LastUpdated: ts}
	sesR := epFor(eps, ocpi.ModSessions, ocpi.Receiver)
	step("PUT session", func() error { return ocpi.PutSession(ctx, do, sesR, nil, own, sess) })
	step("PATCH session", func() error {
		return ocpi.PatchSession(ctx, do, sesR, nil, own, "S1", map[string]any{"kwh": 3.0, "last_updated": ts})
	})
	step("GET session (receiver)", func() error {
		s, err := ocpi.Get[ocpi.Session](ctx, do, ocpi.ObjectURL(sesR, own, "S1"), nil)
		if err == nil && s.KWh != 3 {
			err = fmt.Errorf("kwh %v", s.KWh)
		}
		return err
	})
	cdr := ocpi.CDR{CountryCode: "NL", PartyID: "SIG", ID: "C1", StartDateTime: ts, EndDateTime: ts, CdrToken: sess.CdrToken, AuthMethod: ocpi.AuthCommand,
		CdrLocation: ocpi.CdrLocation{ID: "L1", Address: "Demo Street 1", City: "Utrecht", Country: "NLD", Coordinates: ocpi.GeoLocation{Latitude: "52.090736", Longitude: "5.121420"},
			EVSEUID: "L1-E1", EVSEID: "NL*SIG*EL1", ConnectorID: "1", ConnectorStandard: "IEC_62196_T2", ConnectorFormat: ocpi.FormatSocket, ConnectorPowerType: ocpi.PowerAC3Phase},
		Currency: "EUR", ChargingPeriods: []ocpi.ChargingPeriod{{StartDateTime: ts, Dimensions: []ocpi.CdrDimension{{Type: ocpi.DimEnergy, Volume: 3}}}},
		TotalCost: ocpi.Price{ExclVAT: 1.2}, TotalEnergy: 3, TotalTime: 0.5, LastUpdated: ts}
	var cdrLoc string
	step("POST CDR (with a Location header)", func() error {
		loc, err := ocpi.PostCDR(ctx, do, slash(epFor(eps, ocpi.ModCDRs, ocpi.Receiver)), nil, cdr)
		if err == nil && loc == "" {
			err = fmt.Errorf("no Location header")
		}
		cdrLoc = loc
		return err
	})
	peerDefect("GET the CDR at the Location header", "its Location header is not a URL it serves", func() error {
		back, err := ocpi.GetCDR(ctx, do, cdrLoc, nil)
		if err == nil && !strings.EqualFold(back.ID, "C1") {
			err = fmt.Errorf("CDR %q", back.ID)
		}
		if err != nil {
			return fmt.Errorf("Location %q: %w", cdrLoc, err)
		}
		return nil
	})
	tariff := ocpi.Tariff{CountryCode: "NL", PartyID: "SIG", ID: "TA1", Currency: "EUR", LastUpdated: ts,
		Elements: []ocpi.TariffElement{{PriceComponents: []ocpi.PriceComponent{{Type: ocpi.TariffEnergy, Price: 0.3, StepSize: 1}}}}}
	tarR := epFor(eps, ocpi.ModTariffs, ocpi.Receiver)
	step("PUT tariff", func() error { return ocpi.PutTariff(ctx, do, tarR, nil, own, tariff) })
	step("GET tariff (receiver)", func() error {
		_, err := ocpi.Get[ocpi.Tariff](ctx, do, ocpi.ObjectURL(tarR, own, "TA1"), nil)
		return err
	})
	tariff.ID = "TA2"
	step("PUT and DELETE tariff", func() error {
		if err := ocpi.PutTariff(ctx, do, tarR, nil, own, tariff); err != nil {
			return err
		}
		return ocpi.DeleteTariff(ctx, do, tarR, nil, own, "TA2")
	})
	ci := ocpi.ClientInfo{PartyID: "XYZ", CountryCode: "BE", Role: ocpi.RoleCPO, Status: ocpi.ConnConnected, LastUpdated: ts}
	hubR := epFor(eps, ocpi.ModHubClientInfo, ocpi.Receiver)
	step("PUT and GET hub client info", func() error {
		if err := ocpi.PutClientInfo(ctx, do, hubR, ci); err != nil {
			return err
		}
		back, err := ocpi.Get[ocpi.ClientInfo](ctx, do, ocpi.ObjectURL(hubR, ocpi.Party{CountryCode: "BE", PartyID: "XYZ"}), nil)
		if err == nil && back.Status != ocpi.ConnConnected {
			err = fmt.Errorf("status %q", back.Status)
		}
		return err
	})

	// As an eMSP, from the peer's CPO Sender interfaces (the objects
	// pushed above, read back through the peer's schemas).
	locS := epFor(eps, ocpi.ModLocations, ocpi.Sender)
	step("GET locations (one page)", func() error {
		ls, total, err := ocpi.GetLocations(ctx, do, slash(locS), nil, nil)
		if err == nil && (len(ls) != 3 || total != 3) {
			err = fmt.Errorf("%d locations of %d", len(ls), total)
		}
		return err
	})
	peerDefect("GET locations, paged by two (following Link)", "its Link names the enum member VersionNumber.v_2_2_1 in the path", func() error {
		ls, total, err := ocpi.GetLocations(ctx, do, slash(locS), nil, url.Values{"limit": {"2"}})
		if err == nil && (len(ls) != 3 || total != 3) {
			err = fmt.Errorf("%d locations of %d", len(ls), total)
		}
		return err
	})
	step("GET location, EVSE, connector (sender)", func() error {
		var l ocpi.Location
		var e ocpi.EVSE
		var cn ocpi.Connector
		if err := ocpi.GetLocation(ctx, do, locS, nil, &l, "L2"); err != nil {
			return err
		}
		if err := ocpi.GetLocation(ctx, do, locS, nil, &e, "L2", "L2-E1"); err != nil {
			return err
		}
		if err := ocpi.GetLocation(ctx, do, locS, nil, &cn, "L2", "L2-E1", "1"); err != nil {
			return err
		}
		// CiString: the peer answers in lower case.
		if !strings.EqualFold(l.ID, "L2") || !strings.EqualFold(e.UID, "L2-E1") || cn.MaxAmperage != 32 {
			return fmt.Errorf("%q %q %d", l.ID, e.UID, cn.MaxAmperage)
		}
		return nil
	})
	step("GET unknown location is not found", func() error {
		var l ocpi.Location
		if err := ocpi.GetLocation(ctx, do, locS, nil, &l, "L9"); !ocpi.NotFound(err) {
			return fmt.Errorf("got %v", err)
		}
		return nil
	})
	step("GET sessions", func() error {
		ss, _, err := ocpi.GetSessions(ctx, do, slash(epFor(eps, ocpi.ModSessions, ocpi.Sender)), nil, url.Values{"date_from": {"2026-01-01T00:00:00Z"}})
		if err == nil && len(ss) != 1 {
			err = fmt.Errorf("%d sessions", len(ss))
		}
		return err
	})
	peerDefect("PUT charging preferences", "it answers the preferences object in a list, not a ChargingPreferencesResponse", func() error {
		res, err := ocpi.SetChargingPreferences(ctx, do, epFor(eps, ocpi.ModSessions, ocpi.Sender), nil, "S1", ocpi.ChargingPreferences{ProfileType: ocpi.ProfileFast})
		if err == nil && res != ocpi.PrefsAccepted {
			err = fmt.Errorf("answer %q", res)
		}
		return err
	})
	step("GET CDRs", func() error {
		cs, _, err := ocpi.GetCDRs(ctx, do, slash(epFor(eps, ocpi.ModCDRs, ocpi.Sender)), nil, nil)
		if err == nil && len(cs) != 1 {
			err = fmt.Errorf("%d CDRs", len(cs))
		}
		return err
	})
	step("GET tariffs", func() error {
		ts, _, err := ocpi.GetTariffs(ctx, do, slash(epFor(eps, ocpi.ModTariffs, ocpi.Sender)), nil, nil)
		if err == nil && len(ts) != 1 {
			err = fmt.Errorf("%d tariffs", len(ts))
		}
		return err
	})

	// Tokens: as an eMSP to the peer's CPO Receiver, then as a CPO from
	// the peer's eMSP Sender.
	tok := ocpi.Token{CountryCode: "NL", PartyID: "SIG", UID: "T1", Type: ocpi.TokenRFID, ContractID: "NL-SIG-C1", Issuer: "Sigma", Valid: true,
		Whitelist: ocpi.WhitelistAllowed, LastUpdated: ts}
	tokR := epFor(eps, ocpi.ModTokens, ocpi.Receiver)
	step("PUT token", func() error { return ocpi.PutToken(ctx, do, tokR, nil, tok) })
	step("PATCH token", func() error {
		return ocpi.PatchToken(ctx, do, tokR, nil, own, "T1", ocpi.TokenRFID, map[string]any{"issuer": "Sigma 2", "last_updated": ts})
	})
	step("GET token (receiver)", func() error {
		back, err := ocpi.Get[ocpi.Token](ctx, do, ocpi.TokenURL(tokR, own, "T1", ocpi.TokenRFID), nil)
		if err == nil && back.Issuer != "Sigma 2" {
			err = fmt.Errorf("issuer %q", back.Issuer)
		}
		return err
	})
	tokS := epFor(eps, ocpi.ModTokens, ocpi.Sender)
	step("GET tokens", func() error {
		tl, _, err := ocpi.GetTokens(ctx, do, slash(tokS), nil, nil)
		if err == nil && len(tl) != 1 {
			err = fmt.Errorf("%d tokens", len(tl))
		}
		return err
	})
	step("POST token authorize", func() error {
		ai, err := ocpi.AuthorizeToken(ctx, do, tokS, nil, "T1", ocpi.TokenRFID, &ocpi.LocationReferences{LocationID: "L1"})
		if err == nil && ai.Allowed != ocpi.AllowedAllowed {
			err = fmt.Errorf("allowed %q", ai.Allowed)
		}
		return err
	})

	// Commands and charging profiles: the peer answers at once and POSTs
	// the result to our response URL.
	cmdSender := rt.DetailsURL() + "/sender/commands"
	cmdR := epFor(eps, ocpi.ModCommands, ocpi.Receiver)
	if cmdR == "" {
		// The peer serves its CPO commands receiver but leaves it out of
		// its version details; the path is the one its CPO modules use.
		t.Logf("peer deviation: the version details list no commands RECEIVER; using its CPO path")
		cmdR = strings.TrimSuffix(epFor(eps, ocpi.ModLocations, ocpi.Sender), "/locations") + "/commands"
	}
	for i, cmd := range []struct {
		name string
		body any
	}{
		{ocpi.CmdStartSession, ocpi.StartSession{ResponseURL: cmdSender + "/start-1", Token: tok, LocationID: "L1", EVSEUID: "L1-E1", ConnectorID: "1"}},
		{ocpi.CmdStopSession, ocpi.StopSession{ResponseURL: cmdSender + "/stop-2", SessionID: "S1"}},
		{ocpi.CmdReserveNow, ocpi.ReserveNow{ResponseURL: cmdSender + "/reserve-3", Token: tok, ExpiryDate: "2026-10-11T09:00:00Z", ReservationID: "R1", LocationID: "L1"}},
		{ocpi.CmdCancelReservation, ocpi.CancelReservation{ResponseURL: cmdSender + "/cancel-4", ReservationID: "R1"}},
		{ocpi.CmdUnlockConnector, ocpi.UnlockConnector{ResponseURL: cmdSender + "/unlock-5", LocationID: "L1", EVSEUID: "L1-E1", ConnectorID: "1"}},
	} {
		step("POST command "+cmd.name, func() error {
			cr, err := ocpi.SendCommand(ctx, do, cmdR, nil, cmd.name, cmd.body)
			if err == nil && cr.Result != ocpi.CmdRespAccepted {
				err = fmt.Errorf("response %q", cr.Result)
			}
			return err
		})
		k := []string{"start-1", "stop-2", "reserve-3", "cancel-4", "unlock-5"}[i]
		step("CommandResult of "+cmd.name+" at our response URL", func() error { return waitResult(o, "commands/"+k) })
	}
	cpR := epFor(eps, ocpi.ModChargingProfiles, ocpi.Receiver)
	cpSender := rt.DetailsURL() + "/sender/chargingprofiles"
	dur := 3600
	peerDefect("PUT charging profile", "it requires min_charge_rate and one period object (2.2.1: optional min_charging_rate, a list)", func() error {
		r, err := ocpi.SetChargingProfileFor(ctx, do, cpR, nil, "S1", ocpi.SetChargingProfile{ResponseURL: cpSender + "/set-1",
			ChargingProfile: ocpi.ChargingProfile{ChargingRateUnit: ocpi.RateWatts, Duration: &dur, ChargingProfilePeriod: []ocpi.ChargingProfilePeriod{{StartPeriod: 0, Limit: 7400}}}})
		if err == nil && r.Result != ocpi.ProfileRespAccepted {
			err = fmt.Errorf("response %q", r.Result)
		}
		return err
	})
	peerDefect("ChargingProfileResult at our response URL", "no result follows a refused PUT", func() error { return waitResult(o, "profiles/set-1") })
	step("GET active charging profile", func() error {
		r, err := ocpi.GetActiveChargingProfile(ctx, do, cpR, nil, "S1", 900, cpSender+"/get-1")
		if err == nil && r.Result != ocpi.ProfileRespAccepted {
			err = fmt.Errorf("response %q", r.Result)
		}
		return err
	})
	step("ActiveChargingProfileResult at our response URL", func() error { return waitResult(o, "profiles/get-1") })
	step("DELETE charging profile", func() error {
		r, err := ocpi.ClearChargingProfile(ctx, do, cpR, nil, "S1", cpSender+"/clear-1")
		if err == nil && r.Result != ocpi.ProfileRespAccepted {
			err = fmt.Errorf("response %q", r.Result)
		}
		return err
	})
	step("ClearProfileResult at our response URL", func() error { return waitResult(o, "profiles/clear-1") })

	// Credentials: read, update (the peer reads our versions again),
	// delete; then the old token is refused.
	credU := strings.TrimRight(ocpi.EndpointURL(eps, ocpi.ModCredentials), "/") + "/"
	step("GET credentials", func() error {
		cr, err := ocpi.GetCredentials(ctx, do, credU)
		if err == nil && cr.Token != tokenC {
			err = fmt.Errorf("token %q", cr.Token)
		}
		return err
	})
	step("PUT credentials", func() error {
		cr, err := ocpi.UpdateCredentials(ctx, do, credU, ocpi.Credentials{Token: o.tokenB, URL: rt.VersionsURL(), Roles: theirs.Roles[:1]})
		if err == nil {
			tokenC = cr.Token
		}
		return err
	})
	step("DELETE credentials, then a module refuses the token", func() error {
		if err := ocpi.Unregister(ctx, do, credU); err != nil {
			return err
		}
		_, _, err := ocpi.GetLocations(ctx, do, slash(locS), nil, nil)
		if err == nil {
			return fmt.Errorf("the token still works")
		}
		return nil
	})
}

func waitResult(o *ours, k string) error {
	for i := 0; i < 150; i++ {
		if o.result(k) != "" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("no result within 15 s")
}
