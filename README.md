# evroaming

Back-office protocols for electric vehicle charging in pure Go: OCPI 2.2.1, Hubject's OICP 2.3 (the CPO side) and OSCP 2.0; not affiliated with the EVRoaming Foundation, Hubject or the Open Charge Alliance.

[![Go Reference](https://pkg.go.dev/badge/github.com/cB-Sigma/evroaming.svg)](https://pkg.go.dev/github.com/cB-Sigma/evroaming) [![test](https://github.com/cB-Sigma/evroaming/actions/workflows/test.yml/badge.svg)](https://github.com/cB-Sigma/evroaming/actions/workflows/test.yml) [![Licence: Apache-2.0](https://img.shields.io/badge/licence-Apache--2.0-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/go-1.26-00ADD8.svg)](go.mod)

Three packages, one per protocol. `ocpi` is roaming between charge point
operators (CPO) and eMobility service providers (eMSP): every object and
every module of OCPI 2.2.1, a client helper for every call from either
side, and `ocpi.Router`, which serves any set of Sender and Receiver
interfaces an application implements. `oicp` is a CPO's link to Hubject:
every CPO operation of OICP 2.3, outbound over mutual TLS and inbound
through `oicp.Handler`. `oscp` is smart charging capacity between a
Capacity Provider (CP), a Flexibility Provider (FP) and a Capacity
Optimizer (CO): every OSCP 2.0 message, who may send it to whom, and
validation against the Open Charge Alliance's schemas, which are
embedded unmodified. The application keeps its own state (partners,
tokens, sessions); no cgo. Written for SigmaM2M, a oneM2M platform,
whose EV charging adapters are built on it.

## Features

| Feature | Status |
|---|---|
| OCPI 2.2.1 objects of every module (locations with EVSEs, connectors, hours, images, energy mix; sessions and charging preferences; CDRs with signed data; tariffs; tokens; commands; charging profiles; hub client info; credentials) and every enumeration | yes |
| OCPI client calls for every module, both roles (table below), with Base64 credentials tokens, X-Request-ID, X-Correlation-ID, the four OCPI-to/from routing headers, status codes as `*CallError` | yes |
| OCPI paginated lists both ways: `GetAll` / `List` follow `Link` pages; `Paginate` serves date_from, date_to, offset, limit with X-Total-Count, X-Limit and Link | yes |
| OCPI server: `Router` serves versions, version details, credentials and every module's Sender and Receiver interface (16 interfaces), checks tokens, refuses a registration token outside versions and credentials, answers 404 with 2003/2004 for unknown objects, 405 for methods an interface does not offer, 400/2001 for a PATCH without last_updated | yes |
| OCPI credentials handshake: `FetchEndpoints` (2.2.1, or 2.2 spoken to as 2.2.1), `RegisterCredentials`, `UpdateCredentials`, `GetCredentials`, `Unregister` | yes |
| OCPI leniency: a single object wrapped in a list of one is read as the object (`DecodeData`; one peer answers that way), raw (not Base64) tokens on request, `Router.TrailingSlash` for partners that append paths to endpoint URLs | yes |
| OICP 2.3 CPO operations, all twelve of Hubject's CPO OpenAPI (table below), with every field of its schemas | yes |
| OICP: `Client` (mutual TLS is the `http.Client`'s), `HubjectError`, `Retryable`; `Handler` serves remote start/stop and reservation start/stop; identifier checks (`ValidOperatorID`, `ValidEvseID`, `ValidProviderID`, `ValidHotline`) and `NormID` | yes |
| OSCP 2.0: the nine messages of 4.2 with their directions (`CanSend`, `Endpoints`), datatypes, `Validate` (schemas plus rules the schemas lack), `ValidateRawSchema` (the files as published), headers, `NewToken`, `NewRequestID`, `Segments`, `Post` | yes |
| OCPI 2.1.1 and 2.3.0, OICP's EMP side, Plug&Charge certificate handling (OPCP) | no (see Not yet) |

### OCPI 2.2.1: every interface

| Module | Sender (who serves it) | Receiver (who serves it) | Client helpers | `Router` interface |
|---|---|---|---|---|
| versions | both: GET versions, GET details | | `FetchEndpoints` | built in |
| credentials | both: GET, POST, PUT, DELETE (405 on a second POST) | | `RegisterCredentials`, `GetCredentials`, `UpdateCredentials`, `Unregister` | `CredentialsModule` |
| locations | CPO: GET list; GET location, EVSE, connector | eMSP: GET, PUT, PATCH on location, EVSE, connector | `GetLocations`, `GetLocation`, `PutLocation`, `PatchLocation` | `LocationsSender`, `LocationsReceiver` |
| sessions | CPO: GET list; PUT {id}/charging_preferences | eMSP: GET, PUT, PATCH | `GetSessions`, `SetChargingPreferences`, `PutSession`, `PatchSession` | `SessionsSender`, `SessionsReceiver` |
| cdrs | CPO: GET list | eMSP: POST (answer with a Location header), GET at that URL | `GetCDRs`, `PostCDR`, `GetCDR` | `CDRsSender`, `CDRsReceiver` |
| tariffs | CPO: GET list | eMSP: GET, PUT, DELETE | `GetTariffs`, `PutTariff`, `DeleteTariff` | `TariffsSender`, `TariffsReceiver` |
| tokens | eMSP: GET list; POST {uid}/authorize[?type=] | CPO: GET, PUT, PATCH [?type=] | `GetTokens`, `AuthorizeToken`, `PutToken`, `PatchToken`, `TokenURL` | `TokensSender`, `TokensReceiver` |
| commands | eMSP: takes CommandResult at its response_url | CPO: POST START_SESSION, STOP_SESSION, RESERVE_NOW, CANCEL_RESERVATION, UNLOCK_CONNECTOR | `SendCommand`, `PostCommandResult` | `CommandsSender`, `CommandsReceiver` |
| chargingprofiles | eMSP or SCSP: takes the three results at its response_url; PUT {session_id} active profile | CPO: GET ?duration&response_url, PUT, DELETE ?response_url | `GetActiveChargingProfile`, `SetChargingProfileFor`, `ClearChargingProfile`, `PostChargingProfileResult`, `PutActiveChargingProfile` | `ChargingProfilesSender`, `ChargingProfilesReceiver` |
| hubclientinfo | hub: GET list | CPO or eMSP: GET, PUT | `GetClientInfos`, `PutClientInfo` | `HubClientInfoSender`, `HubClientInfoReceiver` |

Any object can also be read with `Get[T]`, written with `Put`, `Patch`
and `Delete`, and any list with `List[T]`; `ObjectURL` builds the
{country_code}/{party_id}/{id} paths. Status codes 1000, 2000 to 2004,
3000 to 3003 and 4000 to 4003 are constants; HTTP 401 is a bad token,
404 an unknown object, 405 a method the interface does not offer.

### OICP 2.3: every CPO operation

Paths below Hubject's base URL (`https://service.hubject.com/api/oicp`,
QA `service-qa.hubject.com`), all POST over mutual TLS.

| Operation | Direction | Path | Here |
|---|---|---|---|
| eRoamingPushEvseData V2.3 (ActionType fullLoad, update, insert, delete) | CPO to Hubject | /evsepush/v23/operators/{operatorID}/data-records | `PushEvseData`, `PathEvseData` |
| eRoamingPushEvseStatus V2.1 | CPO to Hubject | /evsepush/v21/operators/{operatorID}/status-records | `PushEvseStatus`, `PathEvseStatus` |
| eRoamingPushPricingProductData V1.0 | CPO to Hubject | /dynamicpricing/v10/operators/{operatorID}/pricing-products | `PushPricingProductData`, `PathPricing` |
| eRoamingPushEVSEPricing V1.0 | CPO to Hubject | /dynamicpricing/v10/operators/{operatorID}/evse-pricing | `PushEVSEPricing`, `PathEVSEPricing` |
| eRoamingAuthorizeStart V2.1 (answer eRoamingAuthorizationStart) | CPO to Hubject | /charging/v21/operators/{operatorID}/authorize/start | `AuthorizeStart`, `PathAuthStart` |
| eRoamingAuthorizeStop V2.1 (answer eRoamingAuthorizationStop) | CPO to Hubject | /charging/v21/operators/{operatorID}/authorize/stop | `AuthorizeStop`, `PathAuthStop` |
| eRoamingChargeDetailRecord V2.2 | CPO to Hubject | /cdrmgmt/v22/operators/{operatorID}/charge-detail-record | `SendChargeDetailRecord`, `PathCDR` |
| eRoamingChargingNotifications V1.1 (Start, Progress, End, Error) | CPO to Hubject | /notificationmgmt/v11/charging-notifications (no operatorID) | `SendChargingNotification`, `PathNotifications` |
| eRoamingAuthorizeRemoteStart V2.1 | Hubject to CPO | /charging/v21/providers/{providerID}/authorize-remote/start | `Handler.RemoteStart`, `PathRemoteStart` |
| eRoamingAuthorizeRemoteStop V2.1 | Hubject to CPO | /charging/v21/providers/{providerID}/authorize-remote/stop | `Handler.RemoteStop`, `PathRemoteStop` |
| eRoamingAuthorizeRemoteReservationStart V1.1 | Hubject to CPO | /reservation/v11/providers/{providerID}/reservation-start-request | `Handler.ReservationStart`, `PathReservationStart` |
| eRoamingAuthorizeRemoteReservationStop V1.1 | Hubject to CPO | /reservation/v11/providers/{providerID}/reservation-stop-request | `Handler.ReservationStop`, `PathReservationStop` |

The Hubject-to-CPO paths are where the OpenAPI description lists them;
the CPO serves them at the URLs it gives Hubject, and `Handler` matches
them by their endings. All five identification forms are read and
written (RFIDMifareFamily, RFID, QRCode with a hashed PIN,
PlugAndCharge, Remote); Plug&Charge here means the PlugAndCharge
identification and the PnC authentication mode, not Hubject's separate
certificate protocol.

### OSCP 2.0: every message

All POST below a role's base URL (`.../{cp|fp|co}/2.0`), with
`Authorization: Token <token>`, X-Request-ID, X-Correlation-ID and, for
a message split by `Segments`, X-Segment-Index and X-Segment-Count. A
receiver answers 204 at once and answers a request (a Handshake with a
HandshakeAcknowledge, a Register with its own Register) in a separate
POST later; an error answer carries the error body.

| Message (4.2 endpoint) | Directions (`CanSend`) |
|---|---|
| Register (register) | CP→FP, CO→FP, FP→CP, FP→CO |
| Handshake (handshake) | CP→FP, FP→CO |
| HandshakeAcknowledge (handshake_acknowledge; the variant handshake_acknowledgment is accepted by `CanonicalEndpoint`) | FP→CP, CO→FP |
| Heartbeat (heartbeat) | CP→FP, CO→FP, FP→CP, FP→CO |
| UpdateGroupCapacityForecast (update_group_capacity_forecast) | CP→FP, FP→CO, and CO→FP for the Optimum (OSCP-N5 in PROVENANCE.md) |
| AdjustGroupCapacityForecast (adjust_group_capacity_forecast) | FP→CP |
| GroupCapacityComplianceError (group_capacity_compliance_error) | FP→CP |
| UpdateGroupMeasurements (update_group_measurements) | FP→CP |
| UpdateAssetMeasurements (update_asset_measurements) | FP→CO |

The directions are read from clause 4 of the specification; open
implementations differ (one lets any role send the handshake messages),
so `CanSend` is advice for the application, not enforced by `Post`.

## Install

```
go get github.com/cB-Sigma/evroaming@latest
```

Requires Go 1.26 or later (its JSON Schema dependency needs it).

## Quick start

An eMSP registering with a CPO and reading its locations:

```go
import "github.com/cB-Sigma/evroaming/ocpi"

c := &ocpi.Client{From: ocpi.Party{CountryCode: "NL", PartyID: "MSP"}}
var tokenC string
do := func(ctx context.Context, r ocpi.Request) (*ocpi.Reply, error) {
	tok := r.Token
	if tok == "" {
		tok = tokenC
	}
	return c.Send(ctx, r, tok, false)
}
_, eps, err := ocpi.FetchEndpoints(ctx, do, versionsURL, tokenA, nil)
theirs, err := ocpi.RegisterCredentials(ctx, do, ocpi.EndpointURL(eps, ocpi.ModCredentials), tokenA, ours)
tokenC = theirs.Token
locations, total, err := ocpi.GetLocations(ctx, do, ocpi.EndpointURL(eps, ocpi.ModLocations), &cpo, nil)
```

Serving a CPO's interfaces (implement the ones you offer):

```go
rt := &ocpi.Router{BaseURL: "https://cpo.example/ocpi", Self: cpo, Authenticate: lookupToken,
	Credentials: creds, LocationsSender: store, TokensReceiver: store, CommandsReceiver: chargers}
http.Handle("/ocpi/", rt)
```

A CPO pushing an EVSE status to Hubject and serving remote starts:

```go
c := &oicp.Client{BaseURL: "https://service-qa.hubject.com/api/oicp", HTTP: mtlsClient}
_, err := c.PushEvseStatus(ctx, oicp.PushEvseStatus{ActionType: oicp.ActionUpdate, OperatorEvseStatus: status})
http.Handle("/oicp/", &oicp.Handler{RemoteStart: start, RemoteStop: stop})
```

Runnable examples: `ocpi.Example`, `oscp.Example`.
[evroaming-sim](https://github.com/cB-Sigma/evroaming-sim) runs these
packages against simulated partners.

## Documentation

API reference: https://pkg.go.dev/github.com/cB-Sigma/evroaming

| Package | What it does |
|---|---|
| `ocpi` | objects and enumerations; `Client.Send`, `Doer`, `Request`, `Reply`, `CallError`, `NotFound`; the module helpers above; `GetAll`, `List`, `Get`, `Put`, `Patch`, `Delete`, `Exchange`, `DecodeData`; `Router` and the module interfaces; server helpers `WriteReply`, `TokenCandidates`, `SetRequestIDs`, `SetRoutingHeaders`, `FromHeaders`, `Paginate`, `Decode`, `SplitPath` |
| `oicp` | messages, status codes, identifier checks; `Client` (`Post`, `PostAck` and one method per CPO operation), `HubjectError`, `Retryable`, `StatusText`; `Handler`, `Operation`, `NewAck`, `WriteAck`; the `Path...` functions |
| `oscp` | messages and datatypes; `Endpoints`, `EndpointByName`, `CanonicalEndpoint`, `CanSend`; `Validate`, `ValidateSchema`, `ValidateRawSchema`, `SchemaFiles`, `CheckForecastSet`, `Signed`; headers, `NewToken`, `NewRequestID`, `Segments`, `Post` |
| `oscp/schemas` | the OCA's OSCP 2.0 JSON schemas, unmodified (CC BY-ND 4.0), with SHA256SUMS |
| `interop` (nested module, tests only) | this library against extrawest_ocpi (OCPI) and pyoscp (OSCP) in containers |
| `oicpspec` (nested module, tests only) | every OICP message against Hubject's OpenAPI description, fetched at a pinned commit |

Changes per version are in [CHANGELOG.md](CHANGELOG.md).

## Testing

```
go test -race ./...
cd interop && SIGMA_EVROAMING_INTEROP=1 go test -v ./...   # needs Docker
cd oicpspec && SIGMA_EVROAMING_OICP_SPEC=1 go test -v ./... # needs network (or SIGMA_EVROAMING_OICP_DIR=<checkout>)
```

The unit tests cover the client, the router, pagination, token
handling, the handshake, OICP's errors and identifiers and OSCP's
schemas and directions. Tests that start simulators are in
evroaming-sim.

### Interop tested against

- extrawest_ocpi 2025.7.16 (MIT, github.com/extrawest/extrawest_ocpi),
  2026-10-11, python:3.11.9-slim in Docker Desktop on arm64, serving
  the CPO and the eMSP role from one in-memory store (`interop`).
  Passed, 42 steps: the credentials handshake (the peer read our
  versions and details from `ocpi.Router`), GET, PUT and DELETE of
  credentials and the old token refused afterwards; as a CPO, PUT of
  three locations, an EVSE, PATCH and GET of a location, PUT, PATCH
  and GET of a session, POST of a CDR with its Location header, PUT,
  GET and DELETE of tariffs, PUT and GET of client info; as an eMSP,
  the location list, a location, EVSE and connector, an unknown
  location (404), the session, CDR and tariff lists; PUT, PATCH and
  GET of a token at the CPO, the eMSP's token list and a real-time
  authorisation; all five commands with their CommandResult POSTed
  back to our `Router`; GET and DELETE of a charging profile with the
  ActiveChargingProfileResult and ClearProfileResult POSTed back. Every
  object we sent passed the peer's own schemas. The peer deviates from
  2.2.1 in ways the test records and does not count as passes: its
  single objects come wrapped in a list of one (read through
  `DecodeData`); its CDR Location header and pagination Link name
  Python enum members in the path and always say https, so the CDR
  cannot be read back there and the second page of a list is not
  found; its charging preferences answer is the preferences object,
  not a ChargingPreferencesResponse; its PUT charging profile requires
  min_charge_rate and a single period object (2.2.1: optional
  min_charging_rate and a list), so ours is refused; its version
  details leave out its commands receiver; it requires energy_mix's
  supplier_name and energy_product_name, which are optional in 2.2.1.
- Hubject's OICP 2.3 CPO OpenAPI description
  (github.com/hubject/oicp-cpo-2.3-api-doc, commit f5a8e2f, 2026-05-08;
  `oicpspec`), 2026-10-11: 33 requests built by this package against
  the 15 request schemas, the acknowledgements `Handler` writes and the
  authorisation answers against the response schemas, every example of
  the description read without dropping a member, and the twelve
  operation paths: all passed (three deliberately invalid messages
  were refused). This is the description, not Hubject's service.
- pyoscp 0.1.0 (MIT, github.com/NOWUM/pyoscp), 2026-10-11,
  python:3.11.9-slim in Docker Desktop on arm64, as a Capacity Provider
  (`interop`), with this library as the Flexibility Provider: Register
  both ways (the peer's own Register was valid against the OCA schema
  and carried the correlation ID), then HandshakeAcknowledge (on the
  peer's handshake_acknowledgment path), AdjustGroupCapacityForecast,
  GroupCapacityComplianceError and UpdateGroupMeasurements, accepted by
  the peer's models and handled; a malformed forecast was refused with
  400 and an unknown token with 403. Deviations of the peer: it reads
  offline_mode_at as "YYYY-MM-DD hh:mm:ss", so our RFC 3339 heartbeat
  gets a 500, and its own heartbeat fails our schema check; it
  re-registers only when the first Register has no X-Correlation-ID
  (as 4.3.1 implies); its models mark no field as required, so they
  check types, not completeness. pyoscp implements few of the other
  directions, so only FP-to-CP was run.
- OSCP 2.0 otherwise: every message is validated against the OCA's
  schemas in the unit tests.
- Not yet: a real roaming platform, Hubject's QA service (needs a
  Hubject contract and certificate), an OICP peer, and the CP-to-FP,
  FP-to-CO and CO-to-FP directions of OSCP against another
  implementation.
- The library and its simulators are written by the same author, so
  their agreement is not independent evidence; the runs above are.

## Not yet

| What | Why |
|---|---|
| OCPI 2.1.1 and 2.3.0 objects | 2.2.1 is what partners and hubs ask for; a 2.2 partner is spoken to as 2.2.1 (2.2.1 deprecates 2.2), and raw 2.1.1-style tokens are accepted |
| Hub routing as a hub (forwarding between parties) | the routing headers are sent, read and answered, and `Router` passes them to modules; forwarding is an application's |
| OICP EMP side | the module is for a CPO; the EMP operations are a different OpenAPI description |
| Hubject's Plug&Charge certificate protocol (OPCP) | a separate protocol and service; the PlugAndCharge identification of OICP is supported |
| A run against Hubject's QA service | needs a Hubject contract and client certificate |
| An OICP peer (WWCP_OICP, .NET) and a fuller OSCP peer (SolarNetwork's FP and CP simulator, Java, GPL-2.0, as a separate container) | not run yet; OICP is checked against Hubject's OpenAPI description instead, and OSCP's FP-to-CP messages against pyoscp |

## Related repositories

- [evroaming-sim](https://github.com/cB-Sigma/evroaming-sim): simulated
  OCPI partners of every role, a Hubject-like OICP platform and OSCP
  parties, as Go packages and commands.
- [jsonschema](https://github.com/cB-Sigma/jsonschema): the JSON Schema
  layer `oscp` validates with.

## Status

v0.x: the API may change before v1.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md) to report a vulnerability privately.

## Licence

Apache-2.0 (see [LICENSE](LICENSE) and [NOTICE](NOTICE)), except the
OCA schemas in `oscp/schemas`, which are CC BY-ND 4.0 and unmodified.
Copyright 2026 Abhinav Gautam. OCPI is a protocol of the EVRoaming
Foundation, OICP and Hubject are trademarks of Hubject GmbH, OSCP is a
protocol of the Open Charge Alliance; the names are used only to say
which protocols this module speaks.
