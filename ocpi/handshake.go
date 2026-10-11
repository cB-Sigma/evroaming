package ocpi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Deviation is told about a partner's answer that is off the
// specification but still used (a code such as "ocpi.versionVariant" and
// a message). nil ignores them.
type Deviation func(code, msg string)

// FetchEndpoints reads a partner's versions at versionsURL with token,
// picks 2.2.1 (else 2.2, spoken to as 2.2.1) and reads that version's
// endpoints (the first steps of the credentials handshake, 7.1.1). A
// partner without either version is a *CallError with
// StatusUnsupportedVer.
func FetchEndpoints(ctx context.Context, do Doer, versionsURL, token string, dev Deviation) (string, []Endpoint, error) {
	if dev == nil {
		dev = func(string, string) {}
	}
	rep, err := do(ctx, Request{Method: http.MethodGet, URL: versionsURL, Token: token})
	if err != nil {
		return "", nil, fmt.Errorf("reading the versions: %w", err)
	}
	var vs []VersionInfo
	if err := json.Unmarshal(rep.Envelope.Data, &vs); err != nil {
		// A single object instead of a list.
		var one VersionInfo
		if json.Unmarshal(rep.Envelope.Data, &one) != nil {
			return "", nil, fmt.Errorf("the versions are not a list: %w", err)
		}
		vs = []VersionInfo{one}
		dev("ocpi.messageDeviation", "the versions endpoint answered one object instead of a list")
	}
	var pick VersionInfo
	for _, v := range vs {
		if v.Version == Version {
			pick = v
		}
	}
	if pick.URL == "" {
		for _, v := range vs {
			if v.Version == "2.2" {
				pick = v
				dev("ocpi.versionVariant", "the partner offers OCPI 2.2, not 2.2.1; it is spoken to as 2.2.1")
			}
		}
	}
	if pick.URL == "" {
		var have []string
		for _, v := range vs {
			have = append(have, v.Version)
		}
		return "", nil, &CallError{Code: StatusUnsupportedVer, Message: "the partner offers no OCPI 2.2.1 (it offers " + strings.Join(have, ", ") + ")"}
	}
	rep, err = do(ctx, Request{Method: http.MethodGet, URL: pick.URL, Token: token})
	if err != nil {
		return "", nil, fmt.Errorf("reading the version details: %w", err)
	}
	var det VersionDetails
	if err := json.Unmarshal(rep.Envelope.Data, &det); err != nil {
		if DecodeData(rep.Envelope.Data, &det) != nil {
			return "", nil, fmt.Errorf("the version details: %w", err)
		}
		dev("ocpi.messageDeviation", "the version details are a list of one object instead of an object")
	}
	return pick.Version, det.Endpoints, nil
}

// EndpointURL returns the URL of module in eps ("" when it is not listed).
// The last entry wins when a module is listed for more than one role.
func EndpointURL(eps []Endpoint, module string) string {
	u := ""
	for _, e := range eps {
		if e.Identifier == module {
			u = e.URL
		}
	}
	return u
}

// MutualModules says whether eps has a functional module a CPO or eMSP
// works with (or hubclientinfo).
func MutualModules(eps []Endpoint) bool {
	for _, e := range eps {
		switch e.Identifier {
		case ModLocations, ModSessions, ModCDRs, ModTokens, ModCommands, ModTariffs, ModHubClientInfo:
			return true
		}
	}
	return false
}

// RoleList describes roles for a log line ("CPO NL/ABC, EMSP NL/XYZ").
func RoleList(rs []CredentialsRole) string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Role+" "+Upper(r.CountryCode)+"/"+Upper(r.PartyID))
	}
	return strings.Join(out, ", ")
}

// Endpoints lists the endpoints we serve below detailsURL (the version
// details URL): credentials and hubclientinfo always, and the functional
// modules of the CPO and eMSP roles, each in the interface role that
// party role takes.
func Endpoints(detailsURL string, cpo, emsp bool) []Endpoint {
	b := detailsURL
	eps := []Endpoint{{Identifier: ModCredentials, Role: Sender, URL: b + "/" + ModCredentials},
		{Identifier: ModHubClientInfo, Role: Receiver, URL: b + "/" + ModHubClientInfo}}
	if cpo {
		c := b + "/cpo/"
		eps = append(eps,
			Endpoint{ModLocations, Sender, c + ModLocations}, Endpoint{ModSessions, Sender, c + ModSessions},
			Endpoint{ModCDRs, Sender, c + ModCDRs}, Endpoint{ModTariffs, Sender, c + ModTariffs},
			Endpoint{ModTokens, Receiver, c + ModTokens}, Endpoint{ModCommands, Receiver, c + ModCommands})
	}
	if emsp {
		e := b + "/emsp/"
		eps = append(eps,
			Endpoint{ModLocations, Receiver, e + ModLocations}, Endpoint{ModSessions, Receiver, e + ModSessions},
			Endpoint{ModCDRs, Receiver, e + ModCDRs}, Endpoint{ModTariffs, Receiver, e + ModTariffs},
			Endpoint{ModTokens, Sender, e + ModTokens}, Endpoint{ModCommands, Sender, e + ModCommands})
	}
	return eps
}
