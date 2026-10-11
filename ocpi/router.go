// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Router serves a party's OCPI 2.2.1 endpoints: the versions and version
// details endpoints, credentials, and the Sender and Receiver interfaces
// of every module an application implements. It checks the credentials
// token, reads the routing headers, copies X-Request-ID and
// X-Correlation-ID, sets the routing headers of the answer, writes the
// response envelope and paginates lists; the application implements one
// interface per module and role (LocationsSender, TokensReceiver, ...)
// and keeps its own state.
//
// Paths below BaseURL: /versions, /2.2.1 (details), /2.2.1/credentials,
// /2.2.1/{sender|receiver}/{module}/... . Mount the Router at BaseURL's
// path (strip any prefix before it).
type Router struct {
	// BaseURL is the public URL the paths are below (no trailing slash).
	BaseURL string
	// Self is our party, used in the routing headers of answers.
	Self Party
	// Authenticate takes the credentials token of a request (decoded as
	// TokenCandidates reads it) and returns the partner it belongs to;
	// ok false answers 401. registration is true for the token A of a
	// party that has not registered yet: such a token may only reach the
	// versions, details and credentials endpoints.
	Authenticate func(r *http.Request, token string) (partner any, registration, ok bool)
	// Credentials serves the credentials module; nil leaves it out.
	Credentials CredentialsModule
	// The modules, by interface. A nil one is not offered.
	LocationsSender          LocationsSender
	LocationsReceiver        LocationsReceiver
	SessionsSender           SessionsSender
	SessionsReceiver         SessionsReceiver
	CDRsSender               CDRsSender
	CDRsReceiver             CDRsReceiver
	TariffsSender            TariffsSender
	TariffsReceiver          TariffsReceiver
	TokensSender             TokensSender
	TokensReceiver           TokensReceiver
	CommandsReceiver         CommandsReceiver
	CommandsSender           CommandsSender
	ChargingProfilesReceiver ChargingProfilesReceiver
	ChargingProfilesSender   ChargingProfilesSender
	HubClientInfoReceiver    HubClientInfoReceiver
	HubClientInfoSender      HubClientInfoSender
	// PageLimit is the most objects a list page holds (0: 100).
	PageLimit int
	// Now is the clock (nil: time.Now).
	Now func() time.Time
	// TrailingSlash publishes module endpoint URLs ending in "/", for
	// partners that append object paths to them without a separator.
	// The Router serves both forms either way.
	TrailingSlash bool
}

// Call is one request to a module, as the Router hands it over.
type Call struct {
	Request *http.Request
	// Partner is what Authenticate returned.
	Partner any
	// From and To are the routing headers' parties (nil when absent).
	From, To *Party
	// Query is the request's query.
	Query url.Values
}

// ErrNotFound is the error a module returns for an unknown object: the
// Router answers 404 with status code 2003 (locations) or 2004 (tokens)
// or 2000. A *CallError chooses the HTTP status and code itself; any
// other error is answered with 500 and code 3000.
var ErrNotFound = errors.New("ocpi: unknown object")

// CredentialsModule serves the credentials module (7.2): Get returns
// ours; Register (POST) and Update (PUT) take the partner's and return
// ours (with a new token for the partner); Unregister (DELETE).
type CredentialsModule interface {
	GetCredentials(ctx context.Context, c *Call) (Credentials, error)
	Register(ctx context.Context, c *Call, theirs Credentials) (Credentials, error)
	UpdateCredentials(ctx context.Context, c *Call, theirs Credentials) (Credentials, error)
	Unregister(ctx context.Context, c *Call) error
}

// LocationsSender is a CPO's Sender interface (8.2.1). Locations lists
// every location the caller may see with last_updated; Location returns
// a *Location, *EVSE or *Connector for one, two or three IDs.
type LocationsSender interface {
	Locations(ctx context.Context, c *Call) ([]Item, error)
	Location(ctx context.Context, c *Call, ids ...string) (any, error)
}

// LocationsReceiver is an eMSP's Receiver interface (8.2.2); ids are the
// location ID and, for an EVSE or connector, its uid and ID. Put gets a
// Location, EVSE or Connector as the depth of ids says; Patch gets the
// changed members.
type LocationsReceiver interface {
	GetLocation(ctx context.Context, c *Call, owner Party, ids ...string) (any, error)
	PutLocation(ctx context.Context, c *Call, owner Party, v any, ids ...string) error
	PatchLocation(ctx context.Context, c *Call, owner Party, fields map[string]any, ids ...string) error
}

// SessionsSender is a CPO's Sender interface (9.2.1).
type SessionsSender interface {
	Sessions(ctx context.Context, c *Call) ([]Item, error)
	// SetChargingPreferences returns a ChargingPreferencesResponse value.
	SetChargingPreferences(ctx context.Context, c *Call, sessionID string, p ChargingPreferences) (string, error)
}

// SessionsReceiver is an eMSP's Receiver interface (9.2.2).
type SessionsReceiver interface {
	GetSession(ctx context.Context, c *Call, owner Party, id string) (Session, error)
	PutSession(ctx context.Context, c *Call, owner Party, s Session) error
	PatchSession(ctx context.Context, c *Call, owner Party, id string, fields map[string]any) error
}

// CDRsSender is a CPO's Sender interface (10.2.1).
type CDRsSender interface {
	CDRs(ctx context.Context, c *Call) ([]Item, error)
}

// CDRsReceiver is an eMSP's Receiver interface (10.2.2). After PostCDR
// the Router answers with a Location header at
// {endpoint}/{country_code}/{party_id}/{id} of the CDR, where GetCDR
// reads it.
type CDRsReceiver interface {
	PostCDR(ctx context.Context, c *Call, cdr CDR) error
	GetCDR(ctx context.Context, c *Call, owner Party, id string) (CDR, error)
}

// TariffsSender is a CPO's Sender interface (11.2.1).
type TariffsSender interface {
	Tariffs(ctx context.Context, c *Call) ([]Item, error)
}

// TariffsReceiver is an eMSP's Receiver interface (11.2.2).
type TariffsReceiver interface {
	GetTariff(ctx context.Context, c *Call, owner Party, id string) (Tariff, error)
	PutTariff(ctx context.Context, c *Call, owner Party, t Tariff) error
	DeleteTariff(ctx context.Context, c *Call, owner Party, id string) error
}

// TokensSender is an eMSP's Sender interface (12.2.1).
type TokensSender interface {
	Tokens(ctx context.Context, c *Call) ([]Item, error)
	// Authorize answers a real-time authorisation; loc is nil when the
	// request has no body.
	Authorize(ctx context.Context, c *Call, uid, tokenType string, loc *LocationReferences) (AuthorizationInfo, error)
}

// TokensReceiver is a CPO's Receiver interface (12.2.2); tokenType is the
// type query parameter (RFID when absent).
type TokensReceiver interface {
	GetToken(ctx context.Context, c *Call, owner Party, uid, tokenType string) (Token, error)
	PutToken(ctx context.Context, c *Call, owner Party, t Token) error
	PatchToken(ctx context.Context, c *Call, owner Party, uid, tokenType string, fields map[string]any) error
}

// CommandsReceiver is a CPO's Receiver interface (13.2.1). body is a
// *StartSession, *StopSession, *ReserveNow, *CancelReservation or
// *UnlockConnector.
type CommandsReceiver interface {
	Command(ctx context.Context, c *Call, command string, body any) (CommandResponse, error)
}

// CommandsSender is an eMSP's Sender interface (13.2.2): it takes the
// CommandResult POSTed to a response_url below the endpoint; path is
// what follows the endpoint URL.
type CommandsSender interface {
	CommandResult(ctx context.Context, c *Call, path []string, r CommandResult) error
}

// ChargingProfilesReceiver is a CPO's Receiver interface (14.2.1).
type ChargingProfilesReceiver interface {
	GetActiveProfile(ctx context.Context, c *Call, sessionID string, duration int, responseURL string) (ChargingProfileResponse, error)
	SetProfile(ctx context.Context, c *Call, sessionID string, p SetChargingProfile) (ChargingProfileResponse, error)
	ClearProfile(ctx context.Context, c *Call, sessionID, responseURL string) (ChargingProfileResponse, error)
}

// ChargingProfilesSender is the Sender interface of a party that sends
// profiles (14.2.2): results POSTed to its response_urls (result is a
// raw ActiveChargingProfileResult, ChargingProfileResult or
// ClearProfileResult; path tells which request it answers) and active
// profiles the CPO PUTs.
type ChargingProfilesSender interface {
	ProfileResult(ctx context.Context, c *Call, path []string, result json.RawMessage) error
	PutActiveProfile(ctx context.Context, c *Call, sessionID string, p ActiveChargingProfile) error
}

// HubClientInfoReceiver is the Receiver interface of a party connected to
// a hub (15.2.1).
type HubClientInfoReceiver interface {
	GetClientInfo(ctx context.Context, c *Call, party Party) (ClientInfo, error)
	PutClientInfo(ctx context.Context, c *Call, ci ClientInfo) error
}

// HubClientInfoSender is a hub's Sender interface (15.2.2).
type HubClientInfoSender interface {
	ClientInfos(ctx context.Context, c *Call) ([]Item, error)
}

// route is one module interface the Router serves; iface names the Go
// interface, so that one value implementing several is dispatched by
// the route, not by its type.
type route struct {
	module, role, iface string
	impl                any
}

func (rt *Router) routes() []route {
	all := []route{
		{ModLocations, Sender, "LocationsSender", rt.LocationsSender}, {ModLocations, Receiver, "LocationsReceiver", rt.LocationsReceiver},
		{ModSessions, Sender, "SessionsSender", rt.SessionsSender}, {ModSessions, Receiver, "SessionsReceiver", rt.SessionsReceiver},
		{ModCDRs, Sender, "CDRsSender", rt.CDRsSender}, {ModCDRs, Receiver, "CDRsReceiver", rt.CDRsReceiver},
		{ModTariffs, Sender, "TariffsSender", rt.TariffsSender}, {ModTariffs, Receiver, "TariffsReceiver", rt.TariffsReceiver},
		{ModTokens, Sender, "TokensSender", rt.TokensSender}, {ModTokens, Receiver, "TokensReceiver", rt.TokensReceiver},
		{ModCommands, Receiver, "CommandsReceiver", rt.CommandsReceiver}, {ModCommands, Sender, "CommandsSender", rt.CommandsSender},
		{ModChargingProfiles, Receiver, "ChargingProfilesReceiver", rt.ChargingProfilesReceiver},
		{ModChargingProfiles, Sender, "ChargingProfilesSender", rt.ChargingProfilesSender},
		{ModHubClientInfo, Receiver, "HubClientInfoReceiver", rt.HubClientInfoReceiver},
		{ModHubClientInfo, Sender, "HubClientInfoSender", rt.HubClientInfoSender},
	}
	var out []route
	for _, r := range all {
		if r.impl != nil {
			out = append(out, r)
		}
	}
	return out
}

// VersionsURL is the public URL of our versions endpoint.
func (rt *Router) VersionsURL() string { return rt.BaseURL + "/versions" }

// DetailsURL is the public URL of our 2.2.1 version details.
func (rt *Router) DetailsURL() string { return rt.BaseURL + "/" + Version }

// Endpoints lists the endpoints the Router serves, for the version
// details.
func (rt *Router) Endpoints() []Endpoint {
	var eps []Endpoint
	if rt.Credentials != nil {
		eps = append(eps, Endpoint{Identifier: ModCredentials, Role: Sender, URL: rt.DetailsURL() + "/" + ModCredentials})
	}
	for _, r := range rt.routes() {
		u := rt.endpointURL(r.module, r.role)
		if rt.TrailingSlash {
			u += "/"
		}
		eps = append(eps, Endpoint{Identifier: r.module, Role: r.role, URL: u})
	}
	return eps
}

func (rt *Router) endpointURL(module, role string) string {
	return rt.DetailsURL() + "/" + strings.ToLower(role) + "/" + module
}

func (rt *Router) now() time.Time {
	if rt.Now != nil {
		return rt.Now()
	}
	return time.Now()
}

func (rt *Router) reply(w http.ResponseWriter, data any) {
	WriteReply(w, http.StatusOK, StatusOK, "", data, rt.now())
}

func (rt *Router) fail(w http.ResponseWriter, err error, notFound int) {
	var ce *CallError
	switch {
	case errors.As(err, &ce):
		h := ce.HTTP
		if h == 0 {
			h = http.StatusOK
		}
		code := ce.Code
		if code == 0 {
			code = StatusClientError
		}
		WriteReply(w, h, code, ce.Message, nil, rt.now())
	case errors.Is(err, ErrNotFound):
		WriteReply(w, http.StatusNotFound, notFound, "unknown object", nil, rt.now())
	default:
		WriteReply(w, http.StatusInternalServerError, StatusServerError, err.Error(), nil, rt.now())
	}
}

func (rt *Router) bad(w http.ResponseWriter, msg string) {
	WriteReply(w, http.StatusBadRequest, StatusInvalidParams, msg, nil, rt.now())
}

// ServeHTTP routes a request.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetRequestIDs(w, r)
	SetRoutingHeaders(w, r, rt.Self)
	cands, ok := TokenCandidates(r.Header.Get("Authorization"))
	var partner any
	var registration, found bool
	if ok && rt.Authenticate != nil {
		for _, c := range cands {
			if partner, registration, found = rt.Authenticate(r, c.Token); found {
				break
			}
		}
	}
	if !found {
		WriteReply(w, http.StatusUnauthorized, StatusClientError, "unknown or missing credentials token", nil, rt.now())
		return
	}
	call := &Call{Request: r, Partner: partner, Query: r.URL.Query()}
	if p, ok := FromHeaders(r); ok {
		call.From = &p
	}
	if cc, pid := r.Header.Get("OCPI-to-country-code"), r.Header.Get("OCPI-to-party-id"); cc != "" && pid != "" {
		call.To = &Party{CountryCode: Upper(cc), PartyID: Upper(pid)}
	}
	seg := SplitPath(r.URL.Path)
	if base, err := url.Parse(rt.BaseURL); err == nil {
		bs := SplitPath(base.Path)
		if len(seg) >= len(bs) && strings.Join(seg[:len(bs)], "/") == strings.Join(bs, "/") {
			seg = seg[len(bs):]
		}
	}
	switch {
	case len(seg) == 1 && seg[0] == "versions" && r.Method == http.MethodGet:
		rt.reply(w, []VersionInfo{{Version: Version, URL: rt.DetailsURL()}})
		return
	case len(seg) == 1 && seg[0] == Version && r.Method == http.MethodGet:
		rt.reply(w, VersionDetails{Version: Version, Endpoints: rt.Endpoints()})
		return
	case len(seg) == 2 && seg[0] == Version && seg[1] == ModCredentials && rt.Credentials != nil:
		rt.credentials(w, r, call)
		return
	case len(seg) == 1 && seg[0] == "versions", len(seg) == 1 && seg[0] == Version:
		WriteReply(w, http.StatusMethodNotAllowed, StatusClientError, "method not allowed", nil, rt.now())
		return
	}
	if registration {
		WriteReply(w, http.StatusUnauthorized, StatusClientError, "register first: this token only reaches versions and credentials", nil, rt.now())
		return
	}
	if len(seg) < 3 || seg[0] != Version {
		WriteReply(w, http.StatusNotFound, StatusClientError, "no such endpoint", nil, rt.now())
		return
	}
	role := strings.ToUpper(seg[1])
	for _, rr := range rt.routes() {
		if rr.module == seg[2] && rr.role == role {
			rt.module(w, r, call, rr, seg[3:])
			return
		}
	}
	WriteReply(w, http.StatusNotFound, StatusClientError, "no such endpoint", nil, rt.now())
}

func (rt *Router) credentials(w http.ResponseWriter, r *http.Request, c *Call) {
	ctx := r.Context()
	var out Credentials
	var err error
	switch r.Method {
	case http.MethodGet:
		out, err = rt.Credentials.GetCredentials(ctx, c)
	case http.MethodPost, http.MethodPut:
		var in Credentials
		if derr := Decode(r, &in); derr != nil {
			rt.bad(w, derr.Error())
			return
		}
		if r.Method == http.MethodPost {
			out, err = rt.Credentials.Register(ctx, c, in)
		} else {
			out, err = rt.Credentials.UpdateCredentials(ctx, c, in)
		}
	case http.MethodDelete:
		if err = rt.Credentials.Unregister(ctx, c); err == nil {
			rt.reply(w, nil)
			return
		}
	default:
		WriteReply(w, http.StatusMethodNotAllowed, StatusClientError, "method not allowed", nil, rt.now())
		return
	}
	if err != nil {
		rt.fail(w, err, StatusClientError)
		return
	}
	rt.reply(w, out)
}

// list answers a paginated list request.
func (rt *Router) list(w http.ResponseWriter, r *http.Request, items []Item, err error, self string) {
	if err != nil {
		rt.fail(w, err, StatusClientError)
		return
	}
	limit := rt.PageLimit
	if limit <= 0 {
		limit = 100
	}
	page, perr := Paginate(w.Header(), r, items, limit, self)
	if perr != nil {
		rt.bad(w, perr.Error())
		return
	}
	rt.reply(w, page)
}

func owner(seg []string) (Party, []string, bool) {
	if len(seg) < 2 {
		return Party{}, nil, false
	}
	return Party{CountryCode: Upper(seg[0]), PartyID: Upper(seg[1])}, seg[2:], true
}

func decodeFields(r *http.Request) (map[string]any, error) {
	var f map[string]any
	if err := Decode(r, &f); err != nil {
		return nil, err
	}
	if _, ok := f["last_updated"]; !ok {
		return nil, errors.New("a PATCH must carry last_updated")
	}
	return f, nil
}

func (rt *Router) module(w http.ResponseWriter, r *http.Request, c *Call, rr route, rest []string) {
	ctx := r.Context()
	self := rt.endpointURL(rr.module, rr.role)
	m := r.Method
	notFound := StatusClientError
	switch rr.module {
	case ModLocations:
		notFound = StatusUnknownLocation
	case ModTokens:
		notFound = StatusUnknownToken
	}
	done := func(data any, err error) {
		if err != nil {
			rt.fail(w, err, notFound)
			return
		}
		rt.reply(w, data)
	}
	switch rr.iface {
	case "LocationsSender":
		impl := rr.impl.(LocationsSender)
		switch {
		case m == http.MethodGet && len(rest) == 0:
			items, err := impl.Locations(ctx, c)
			rt.list(w, r, items, err, self)
		case m == http.MethodGet && len(rest) <= 3:
			done(impl.Location(ctx, c, rest...))
		default:
			rt.notAllowed(w)
		}
	case "LocationsReceiver":
		impl := rr.impl.(LocationsReceiver)
		own, ids, ok := owner(rest)
		if !ok || len(ids) < 1 || len(ids) > 3 {
			rt.bad(w, "the path needs country_code, party_id and location_id")
			return
		}
		switch m {
		case http.MethodGet:
			done(impl.GetLocation(ctx, c, own, ids...))
		case http.MethodPut:
			var v any
			switch len(ids) {
			case 1:
				v = &Location{}
			case 2:
				v = &EVSE{}
			default:
				v = &Connector{}
			}
			if err := Decode(r, v); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutLocation(ctx, c, own, v, ids...))
		case http.MethodPatch:
			f, err := decodeFields(r)
			if err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PatchLocation(ctx, c, own, f, ids...))
		default:
			rt.notAllowed(w)
		}
	case "SessionsSender":
		impl := rr.impl.(SessionsSender)
		switch {
		case m == http.MethodGet && len(rest) == 0:
			items, err := impl.Sessions(ctx, c)
			rt.list(w, r, items, err, self)
		case m == http.MethodPut && len(rest) == 2 && rest[1] == "charging_preferences":
			var p ChargingPreferences
			if err := Decode(r, &p); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(impl.SetChargingPreferences(ctx, c, rest[0], p))
		default:
			rt.notAllowed(w)
		}
	case "SessionsReceiver":
		impl := rr.impl.(SessionsReceiver)
		own, ids, ok := owner(rest)
		if !ok || len(ids) != 1 {
			rt.bad(w, "the path needs country_code, party_id and session_id")
			return
		}
		switch m {
		case http.MethodGet:
			done(impl.GetSession(ctx, c, own, ids[0]))
		case http.MethodPut:
			var s Session
			if err := Decode(r, &s); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutSession(ctx, c, own, s))
		case http.MethodPatch:
			f, err := decodeFields(r)
			if err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PatchSession(ctx, c, own, ids[0], f))
		default:
			rt.notAllowed(w)
		}
	case "CDRsSender":
		impl := rr.impl.(CDRsSender)
		if m != http.MethodGet || len(rest) != 0 {
			rt.notAllowed(w)
			return
		}
		items, err := impl.CDRs(ctx, c)
		rt.list(w, r, items, err, self)
	case "CDRsReceiver":
		impl := rr.impl.(CDRsReceiver)
		switch {
		case m == http.MethodPost && len(rest) == 0:
			var cdr CDR
			if err := Decode(r, &cdr); err != nil {
				rt.bad(w, err.Error())
				return
			}
			if err := impl.PostCDR(ctx, c, cdr); err != nil {
				rt.fail(w, err, notFound)
				return
			}
			w.Header().Set("Location", ObjectURL(self, Party{cdr.CountryCode, cdr.PartyID}, cdr.ID))
			rt.reply(w, nil)
		case m == http.MethodGet && len(rest) == 3:
			own, ids, _ := owner(rest)
			done(impl.GetCDR(ctx, c, own, ids[0]))
		default:
			rt.notAllowed(w)
		}
	case "TariffsSender":
		impl := rr.impl.(TariffsSender)
		if m != http.MethodGet || len(rest) != 0 {
			rt.notAllowed(w)
			return
		}
		items, err := impl.Tariffs(ctx, c)
		rt.list(w, r, items, err, self)
	case "TariffsReceiver":
		impl := rr.impl.(TariffsReceiver)
		own, ids, ok := owner(rest)
		if !ok || len(ids) != 1 {
			rt.bad(w, "the path needs country_code, party_id and tariff_id")
			return
		}
		switch m {
		case http.MethodGet:
			done(impl.GetTariff(ctx, c, own, ids[0]))
		case http.MethodPut:
			var t Tariff
			if err := Decode(r, &t); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutTariff(ctx, c, own, t))
		case http.MethodDelete:
			done(nil, impl.DeleteTariff(ctx, c, own, ids[0]))
		default:
			rt.notAllowed(w)
		}
	case "TokensSender":
		impl := rr.impl.(TokensSender)
		switch {
		case m == http.MethodGet && len(rest) == 0:
			items, err := impl.Tokens(ctx, c)
			rt.list(w, r, items, err, self)
		case m == http.MethodPost && len(rest) == 2 && rest[1] == "authorize":
			var loc *LocationReferences
			if r.ContentLength != 0 {
				var l LocationReferences
				if err := Decode(r, &l); err == nil && l.LocationID != "" {
					loc = &l
				}
			}
			done(impl.Authorize(ctx, c, rest[0], tokenType(c.Query), loc))
		default:
			rt.notAllowed(w)
		}
	case "TokensReceiver":
		impl := rr.impl.(TokensReceiver)
		own, ids, ok := owner(rest)
		if !ok || len(ids) != 1 {
			rt.bad(w, "the path needs country_code, party_id and token_uid")
			return
		}
		switch m {
		case http.MethodGet:
			done(impl.GetToken(ctx, c, own, ids[0], tokenType(c.Query)))
		case http.MethodPut:
			var t Token
			if err := Decode(r, &t); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutToken(ctx, c, own, t))
		case http.MethodPatch:
			f, err := decodeFields(r)
			if err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PatchToken(ctx, c, own, ids[0], tokenType(c.Query), f))
		default:
			rt.notAllowed(w)
		}
	case "CommandsReceiver":
		impl := rr.impl.(CommandsReceiver)
		if m != http.MethodPost || len(rest) != 1 {
			rt.notAllowed(w)
			return
		}
		var body any
		switch rest[0] {
		case CmdStartSession:
			body = &StartSession{}
		case CmdStopSession:
			body = &StopSession{}
		case CmdReserveNow:
			body = &ReserveNow{}
		case CmdCancelReservation:
			body = &CancelReservation{}
		case CmdUnlockConnector:
			body = &UnlockConnector{}
		default:
			WriteReply(w, http.StatusNotFound, StatusClientError, "unknown command", nil, rt.now())
			return
		}
		if err := Decode(r, body); err != nil {
			rt.bad(w, err.Error())
			return
		}
		done(impl.Command(ctx, c, rest[0], body))
	case "CommandsSender":
		impl := rr.impl.(CommandsSender)
		if m != http.MethodPost {
			rt.notAllowed(w)
			return
		}
		var res CommandResult
		if err := Decode(r, &res); err != nil {
			rt.bad(w, err.Error())
			return
		}
		done(nil, impl.CommandResult(ctx, c, rest, res))
	case "ChargingProfilesReceiver":
		impl := rr.impl.(ChargingProfilesReceiver)
		if len(rest) != 1 {
			rt.bad(w, "the path needs session_id")
			return
		}
		switch m {
		case http.MethodGet:
			d, err := parseInt(c.Query.Get("duration"))
			if err != nil || c.Query.Get("response_url") == "" {
				rt.bad(w, "duration and response_url are needed")
				return
			}
			done(impl.GetActiveProfile(ctx, c, rest[0], d, c.Query.Get("response_url")))
		case http.MethodPut:
			var p SetChargingProfile
			if err := Decode(r, &p); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(impl.SetProfile(ctx, c, rest[0], p))
		case http.MethodDelete:
			if c.Query.Get("response_url") == "" {
				rt.bad(w, "response_url is needed")
				return
			}
			done(impl.ClearProfile(ctx, c, rest[0], c.Query.Get("response_url")))
		default:
			rt.notAllowed(w)
		}
	case "ChargingProfilesSender":
		impl := rr.impl.(ChargingProfilesSender)
		switch m {
		case http.MethodPost:
			var raw json.RawMessage
			if err := Decode(r, &raw); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.ProfileResult(ctx, c, rest, raw))
		case http.MethodPut:
			if len(rest) != 1 {
				rt.bad(w, "the path needs session_id")
				return
			}
			var p ActiveChargingProfile
			if err := Decode(r, &p); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutActiveProfile(ctx, c, rest[0], p))
		default:
			rt.notAllowed(w)
		}
	case "HubClientInfoReceiver":
		impl := rr.impl.(HubClientInfoReceiver)
		own, ids, ok := owner(rest)
		if !ok || len(ids) != 0 {
			rt.bad(w, "the path needs country_code and party_id")
			return
		}
		switch m {
		case http.MethodGet:
			done(impl.GetClientInfo(ctx, c, own))
		case http.MethodPut:
			var ci ClientInfo
			if err := Decode(r, &ci); err != nil {
				rt.bad(w, err.Error())
				return
			}
			done(nil, impl.PutClientInfo(ctx, c, ci))
		default:
			rt.notAllowed(w)
		}
	case "HubClientInfoSender":
		impl := rr.impl.(HubClientInfoSender)
		if m != http.MethodGet || len(rest) != 0 {
			rt.notAllowed(w)
			return
		}
		items, err := impl.ClientInfos(ctx, c)
		rt.list(w, r, items, err, self)
	}
}

func (rt *Router) notAllowed(w http.ResponseWriter) {
	WriteReply(w, http.StatusMethodNotAllowed, StatusClientError, "method or path not offered by this interface", nil, rt.now())
}

func tokenType(q url.Values) string {
	if t := q.Get("type"); t != "" {
		return t
	}
	return TokenRFID
}

func parseInt(s string) (int, error) {
	var n int
	if s == "" {
		return 0, errors.New("empty")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(ch-'0')
		if n > 1<<30 {
			return 0, errors.New("too large")
		}
	}
	return n, nil
}
