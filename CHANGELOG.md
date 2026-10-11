# Changelog

All notable changes to this module. Versions follow semantic versioning;
before v1 the API may change in minor versions.

## v0.1.0 (2026-10-11)

First release as a module of its own, split from SigmaM2M.

- OCPI 2.2.1 (`ocpi`): every object and enumeration of the version,
  including the charging profiles module (`ModChargingProfiles`,
  `SetChargingProfile`, `ChargingProfile`, the three result objects),
  charging preferences, signed CDR data, images, energy mix, regular
  and exceptional hours, status schedules and publish tokens.
- OCPI client helpers for every module call from either side
  (`RegisterCredentials` to `PutClientInfo`; see the README), generic
  `Get`, `Put`, `Patch`, `Delete`, `List`, `Exchange`, `ObjectURL`,
  `TokenURL`, and `DecodeData`, which reads a single object wrapped in
  a list of one.
- OCPI `Router`: versions, version details, credentials and the Sender
  and Receiver interfaces of every module behind one interface each,
  with token checks, routing headers, request IDs, pagination and the
  status codes and HTTP statuses of 2.2.1; `TrailingSlash` for
  partners that append paths to endpoint URLs.
- `FetchEndpoints` also reads version details sent as a list of one.
- OICP 2.3 (`oicp`): every field of Hubject's CPO OpenAPI schemas (EVSE
  data record, address, coordinates in all three forms, charge detail
  record with signed metering values and calibration law data,
  charging notifications), every status code, eRoamingAuthorizeRemote
  ReservationStart and Stop (`RemoteReservationStart`,
  `RemoteReservationStop`), the Hubject-to-CPO paths and `Handler`
  that serves them, one `Client` method per CPO operation.
- OSCP 2.0 (`oscp`): `Endpoints`, `CanonicalEndpoint` and
  `EPHandshakeAcknowledgment` (the variant spelling some
  implementations use); comments cite notes in PROVENANCE.md instead of
  an outside register.
- The simulators moved to their own module,
  [evroaming-sim](https://github.com/cB-Sigma/evroaming-sim), with the
  tests that start them.
- Test-only nested modules: `interop` (against extrawest_ocpi 2025.7.16 and pyoscp 0.1.0
  in Docker) and `oicpspec` (against Hubject's OpenAPI
  description at a pinned commit).
