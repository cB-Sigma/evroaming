// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// The calls of every module, from the side that makes them. Each takes
// the Doer of the partner, the URL of the partner's endpoint for the
// module (EndpointURL) and, for functional modules, the party addressed
// (the To of the routing headers; nil leaves them out). Receiver
// endpoints of objects owned by a party take that owner in the path
// (ObjectURL). Answers are decoded from the envelope's data.

// ObjectURL is the URL of one object on a receiver interface:
// base/{country_code}/{party_id}/{ids...} (Locations: location, EVSE
// and connector; Sessions, CDRs on GET, Tariffs, Tokens, ClientInfo).
func ObjectURL(base string, owner Party, ids ...string) string {
	return JoinURL(base, append([]string{owner.CountryCode, owner.PartyID}, ids...)...)
}

// Exchange sends one request and decodes the answer's data into out (nil to
// ignore it). It returns the reply for its headers.
func Exchange(ctx context.Context, do Doer, method, u string, to *Party, body, out any) (*Reply, error) {
	rep, err := do(ctx, Request{Method: method, URL: u, To: to, Body: body})
	if err != nil {
		return rep, err
	}
	if out != nil && len(rep.Envelope.Data) > 0 && string(rep.Envelope.Data) != "null" {
		if err := DecodeData(rep.Envelope.Data, out); err != nil {
			return rep, fmt.Errorf("the answer's data: %w", err)
		}
	}
	return rep, nil
}

// DecodeData decodes an envelope's data into out. Some implementations
// wrap a single object in a list of one; when out is not a slice, such
// a list is read as its element.
func DecodeData(data json.RawMessage, out any) error {
	err := json.Unmarshal(data, out)
	if err == nil {
		return nil
	}
	var one []json.RawMessage
	if json.Unmarshal(data, &one) == nil && len(one) == 1 {
		if json.Unmarshal(one[0], out) == nil {
			return nil
		}
	}
	return err
}

// List reads every page of a sender interface's list (GetAll) and
// decodes the objects.
func List[T any](ctx context.Context, do Doer, u string, to *Party, query url.Values) ([]T, int, error) {
	raw, total, err := GetAll(ctx, do, u, to, query)
	out := make([]T, 0, len(raw))
	for i, r := range raw {
		var v T
		if jerr := json.Unmarshal(r, &v); jerr != nil {
			return out, total, fmt.Errorf("object %d of the list: %w", i+1, jerr)
		}
		out = append(out, v)
	}
	return out, total, err
}

// Get reads one object (GET) into a T.
func Get[T any](ctx context.Context, do Doer, u string, to *Party) (T, error) {
	var v T
	_, err := Exchange(ctx, do, http.MethodGet, u, to, nil, &v)
	return v, err
}

// Put sends a whole object (PUT).
func Put(ctx context.Context, do Doer, u string, to *Party, v any) error {
	_, err := Exchange(ctx, do, http.MethodPut, u, to, v, nil)
	return err
}

// Patch sends the changed members of an object (PATCH); fields must hold
// last_updated (8.2.2.4 and the other modules' PATCH rules).
func Patch(ctx context.Context, do Doer, u string, to *Party, fields map[string]any) error {
	if _, ok := fields["last_updated"]; !ok {
		return fmt.Errorf("a PATCH must carry last_updated")
	}
	_, err := Exchange(ctx, do, http.MethodPatch, u, to, fields, nil)
	return err
}

// Delete removes an object (DELETE: Tariffs, credentials).
func Delete(ctx context.Context, do Doer, u string, to *Party) error {
	_, err := Exchange(ctx, do, http.MethodDelete, u, to, nil, nil)
	return err
}

// Credentials module (7): the registering party POSTs its credentials,
// updates them with PUT, reads the partner's with GET and unregisters
// with DELETE; each answer carries the partner's credentials.

// RegisterCredentials POSTs ours to the partner's credentials endpoint
// (with token A as the Doer's token) and returns the partner's.
func RegisterCredentials(ctx context.Context, do Doer, u, tokenA string, ours Credentials) (Credentials, error) {
	var theirs Credentials
	rep, err := do(ctx, Request{Method: http.MethodPost, URL: u, Token: tokenA, Body: ours})
	if err == nil {
		err = DecodeData(rep.Envelope.Data, &theirs)
	}
	return theirs, err
}

// UpdateCredentials PUTs new credentials and returns the partner's.
func UpdateCredentials(ctx context.Context, do Doer, u string, ours Credentials) (Credentials, error) {
	var theirs Credentials
	_, err := Exchange(ctx, do, http.MethodPut, u, nil, ours, &theirs)
	return theirs, err
}

// GetCredentials reads the partner's credentials.
func GetCredentials(ctx context.Context, do Doer, u string) (Credentials, error) {
	return Get[Credentials](ctx, do, u, nil)
}

// Unregister DELETEs the registration.
func Unregister(ctx context.Context, do Doer, u string) error { return Delete(ctx, do, u, nil) }

// Locations module (8).

// GetLocations reads a CPO's locations (Sender, eMSP side).
func GetLocations(ctx context.Context, do Doer, u string, to *Party, q url.Values) ([]Location, int, error) {
	return List[Location](ctx, do, u, to, q)
}

// GetLocation reads one location, EVSE or connector from a CPO's
// Sender interface: u/{location_id}[/{evse_uid}[/{connector_id}]]. out
// is a *Location, *EVSE or *Connector.
func GetLocation(ctx context.Context, do Doer, u string, to *Party, out any, ids ...string) error {
	_, err := Exchange(ctx, do, http.MethodGet, JoinURL(u, ids...), to, nil, out)
	return err
}

// PutLocation pushes a location, EVSE or connector to an eMSP's Receiver
// interface (CPO side): owner is the CPO, ids the location ID and, for
// an EVSE or connector, its uid and ID.
func PutLocation(ctx context.Context, do Doer, u string, to *Party, owner Party, v any, ids ...string) error {
	return Put(ctx, do, ObjectURL(u, owner, ids...), to, v)
}

// PatchLocation sends changed members of a location, EVSE or connector
// to an eMSP's Receiver interface.
func PatchLocation(ctx context.Context, do Doer, u string, to *Party, owner Party, fields map[string]any, ids ...string) error {
	return Patch(ctx, do, ObjectURL(u, owner, ids...), to, fields)
}

// Sessions module (9).

// GetSessions reads a CPO's sessions (Sender, eMSP side); date_from is
// required by the module.
func GetSessions(ctx context.Context, do Doer, u string, to *Party, q url.Values) ([]Session, int, error) {
	return List[Session](ctx, do, u, to, q)
}

// SetChargingPreferences PUTs a driver's preferences for a session to the
// CPO's Sender interface (u/{session_id}/charging_preferences) and
// returns the ChargingPreferencesResponse value.
func SetChargingPreferences(ctx context.Context, do Doer, u string, to *Party, sessionID string, p ChargingPreferences) (string, error) {
	var res string
	_, err := Exchange(ctx, do, http.MethodPut, JoinURL(u, sessionID, "charging_preferences"), to, p, &res)
	return res, err
}

// PutSession pushes a session to an eMSP's Receiver interface.
func PutSession(ctx context.Context, do Doer, u string, to *Party, owner Party, s Session) error {
	return Put(ctx, do, ObjectURL(u, owner, s.ID), to, s)
}

// PatchSession sends changed members of a session.
func PatchSession(ctx context.Context, do Doer, u string, to *Party, owner Party, sessionID string, fields map[string]any) error {
	return Patch(ctx, do, ObjectURL(u, owner, sessionID), to, fields)
}

// CDRs module (10).

// GetCDRs reads a CPO's CDRs (Sender, eMSP side).
func GetCDRs(ctx context.Context, do Doer, u string, to *Party, q url.Values) ([]CDR, int, error) {
	return List[CDR](ctx, do, u, to, q)
}

// PostCDR posts a CDR to an eMSP's Receiver interface and returns the
// URL the eMSP gives for it (the Location header), where GetCDR reads it.
func PostCDR(ctx context.Context, do Doer, u string, to *Party, c CDR) (string, error) {
	rep, err := Exchange(ctx, do, http.MethodPost, u, to, c, nil)
	if err != nil || rep == nil {
		return "", err
	}
	return rep.Header.Get("Location"), nil
}

// GetCDR reads a CDR back from the URL PostCDR returned.
func GetCDR(ctx context.Context, do Doer, location string, to *Party) (CDR, error) {
	return Get[CDR](ctx, do, location, to)
}

// Tariffs module (11).

// GetTariffs reads a CPO's tariffs (Sender, eMSP side).
func GetTariffs(ctx context.Context, do Doer, u string, to *Party, q url.Values) ([]Tariff, int, error) {
	return List[Tariff](ctx, do, u, to, q)
}

// PutTariff pushes a tariff to an eMSP's Receiver interface.
func PutTariff(ctx context.Context, do Doer, u string, to *Party, owner Party, t Tariff) error {
	return Put(ctx, do, ObjectURL(u, owner, t.ID), to, t)
}

// DeleteTariff removes a tariff from an eMSP's Receiver interface.
func DeleteTariff(ctx context.Context, do Doer, u string, to *Party, owner Party, tariffID string) error {
	return Delete(ctx, do, ObjectURL(u, owner, tariffID), to)
}

// Tokens module (12).

// GetTokens reads an eMSP's tokens (Sender, CPO side).
func GetTokens(ctx context.Context, do Doer, u string, to *Party, q url.Values) ([]Token, int, error) {
	return List[Token](ctx, do, u, to, q)
}

// AuthorizeToken asks an eMSP for real-time authorisation of a token
// (u/{token_uid}/authorize[?type=]); loc may be nil.
func AuthorizeToken(ctx context.Context, do Doer, u string, to *Party, uid, tokenType string, loc *LocationReferences) (AuthorizationInfo, error) {
	target := JoinURL(u, uid, "authorize")
	if tokenType != "" {
		target += "?type=" + url.QueryEscape(tokenType)
	}
	var ai AuthorizationInfo
	var body any
	if loc != nil {
		body = loc
	}
	_, err := Exchange(ctx, do, http.MethodPost, target, to, body, &ai)
	return ai, err
}

// TokenURL is a token on a CPO's Receiver interface:
// u/{country_code}/{party_id}/{token_uid}[?type=].
func TokenURL(u string, owner Party, uid, tokenType string) string {
	t := ObjectURL(u, owner, uid)
	if tokenType != "" {
		t += "?type=" + url.QueryEscape(tokenType)
	}
	return t
}

// PutToken pushes a token to a CPO's Receiver interface (eMSP side).
func PutToken(ctx context.Context, do Doer, u string, to *Party, t Token) error {
	return Put(ctx, do, TokenURL(u, Party{t.CountryCode, t.PartyID}, t.UID, t.Type), to, t)
}

// PatchToken sends changed members of a token.
func PatchToken(ctx context.Context, do Doer, u string, to *Party, owner Party, uid, tokenType string, fields map[string]any) error {
	return Patch(ctx, do, TokenURL(u, owner, uid, tokenType), to, fields)
}

// Commands module (13).

// SendCommand POSTs a command (StartSession, StopSession, ReserveNow,
// CancelReservation, UnlockConnector) to a CPO's Receiver interface
// (u/{command}) and returns its CommandResponse.
func SendCommand(ctx context.Context, do Doer, u string, to *Party, command string, body any) (CommandResponse, error) {
	var cr CommandResponse
	_, err := Exchange(ctx, do, http.MethodPost, JoinURL(u, command), to, body, &cr)
	return cr, err
}

// PostCommandResult POSTs a command's result to the response_url the
// eMSP gave (CPO side).
func PostCommandResult(ctx context.Context, do Doer, responseURL string, to *Party, r CommandResult) error {
	_, err := Exchange(ctx, do, http.MethodPost, responseURL, to, r, nil)
	return err
}

// Charging profiles module (14). The sender of profiles (a smart
// charging service provider or eMSP) calls the CPO's Receiver interface;
// the CPO posts results to the response_url and pushes changed active
// profiles to the sender's interface.

// GetActiveChargingProfile asks for the active profile of a session for
// duration seconds (u/{session_id}?duration=&response_url=).
func GetActiveChargingProfile(ctx context.Context, do Doer, u string, to *Party, sessionID string, duration int, responseURL string) (ChargingProfileResponse, error) {
	q := url.Values{"duration": {strconv.Itoa(duration)}, "response_url": {responseURL}}
	var r ChargingProfileResponse
	_, err := Exchange(ctx, do, http.MethodGet, JoinURL(u, sessionID)+"?"+q.Encode(), to, nil, &r)
	return r, err
}

// SetChargingProfileFor PUTs a profile for a session (u/{session_id}).
func SetChargingProfileFor(ctx context.Context, do Doer, u string, to *Party, sessionID string, p SetChargingProfile) (ChargingProfileResponse, error) {
	var r ChargingProfileResponse
	_, err := Exchange(ctx, do, http.MethodPut, JoinURL(u, sessionID), to, p, &r)
	return r, err
}

// ClearChargingProfile DELETEs a session's profile
// (u/{session_id}?response_url=).
func ClearChargingProfile(ctx context.Context, do Doer, u string, to *Party, sessionID, responseURL string) (ChargingProfileResponse, error) {
	var r ChargingProfileResponse
	_, err := Exchange(ctx, do, http.MethodDelete, JoinURL(u, sessionID)+"?response_url="+url.QueryEscape(responseURL), to, nil, &r)
	return r, err
}

// PostChargingProfileResult POSTs a result (ActiveChargingProfileResult,
// ChargingProfileResult or ClearProfileResult) to a response_url (CPO
// side).
func PostChargingProfileResult(ctx context.Context, do Doer, responseURL string, to *Party, result any) error {
	_, err := Exchange(ctx, do, http.MethodPost, responseURL, to, result, nil)
	return err
}

// PutActiveChargingProfile pushes a session's changed active profile to
// the profile sender's interface (u/{session_id}; CPO side).
func PutActiveChargingProfile(ctx context.Context, do Doer, u string, to *Party, sessionID string, p ActiveChargingProfile) error {
	return Put(ctx, do, JoinURL(u, sessionID), to, p)
}

// Hub client info module (15).

// GetClientInfos reads a hub's list of connected parties (Sender).
func GetClientInfos(ctx context.Context, do Doer, u string, q url.Values) ([]ClientInfo, int, error) {
	return List[ClientInfo](ctx, do, u, nil, q)
}

// PutClientInfo pushes a party's connection status to a Receiver
// interface (hub side).
func PutClientInfo(ctx context.Context, do Doer, u string, ci ClientInfo) error {
	return Put(ctx, do, ObjectURL(u, Party{ci.CountryCode, ci.PartyID}), nil, ci)
}
