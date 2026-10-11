# Provenance

## How the code was written

Written for SigmaM2M, a oneM2M platform, in Go from the published
protocol documents below. No code was ported or copied from another
implementation. The message types, field names, enumerations, status
codes and paths are facts every implementation must use; the code around
them (Go types, the client and server helpers, the router, the
pagination and handshake functions) is our own design. No text of the
documents is reproduced; comments cite clause or section numbers in our
own words.

The packages began inside SigmaM2M's EV charging adapters and moved into
this module on 2026-10-06 without a change in behaviour (files that
moved whole kept their history). The simulators moved out to
evroaming-sim on 2026-10-11, so that the library does not depend on
them. For v0.1.0 (2026-10-11) the following were added from the
documents: the remaining OCPI 2.2.1 objects and enumerations, the
charging profiles module, a client helper for every module call and
`ocpi.Router`; the OICP fields, status codes and operations the first
version lacked (reservation start and stop, `oicp.Handler`, one client
method per operation), each checked against Hubject's OpenAPI
description; `oscp.Endpoints` and `oscp.CanonicalEndpoint`.

## Notes on the OSCP documents

Where the OSCP 2.0 schemas or text contradict each other, the code
follows the reading below; comments refer to these notes.

| Note | Where | Reading |
|---|---|---|
| OSCP-N1 | update-asset-measurements.json | measurements has a $ref next to "type": "array"; under draft-07 the sibling keywords are ignored and every array fails, while clause 5.6 and 4.5.2 make it a list of AssetMeasurement with at least one entry. Corrected in memory. |
| OSCP-N2 | forecasted-block.json | capacity has minimum 0, but generation and fallback generation capacities are negative (2.3, FR.01.06). The minimum is lifted in memory and the sign checked per forecast type in code. |
| OSCP-N3 | required-behaviour.json | measurement_configuration is required by the schema but optional in 5.2, and heartbeat_interval is a number in the schema but an integer of 1 or more in 5.2. In memory the requirement becomes "at least one of the two fields" (as 5.2 asks) and heartbeat_interval an integer of 1 or more; messages built here always carry the list, so the published schema accepts them too. |
| OSCP-N4 | FR.01.08, FR.02.07 | the two requirements disagree on the fallback generation capacity; the code follows FR.02.07, which the example of table 7 also follows. |
| OSCP-N5 | 4.4.1 and use cases 2 and 4 | 4.4.1 lists UpdateGroupCapacityForecast from CP to FP only, while use cases 2 and 4 (3.2.2, 3.3.1) have the CO send its Optimum to the FP with it; `CanSend` allows both. |

## Sources

| Area (files) | Followed | Licence of the source | Taken |
|---|---|---|---|
| OCPI objects, modules, status codes, credentials, pagination, routing headers (ocpi/) | OCPI 2.2.1, EVRoaming Foundation, github.com/ocpi/ocpi tag 2.2.1 (commit 4809735) | CC BY-ND 4.0 | facts only |
| OICP messages, identifier forms, status codes, paths (oicp/) | OICP 2.3 CPO, Hubject, github.com/hubject/oicp (OICP-2.3/OICP 2.3 CPO) and github.com/hubject/oicp-cpo-2.3-api-doc (src/, commit f5a8e2f, 2026-05-08) | CC BY-SA 4.0 | facts only |
| OSCP messages, directions, rules the schemas lack (oscp/) | OSCP 2.0 FINAL 2020-10-12, Open Charge Alliance | CC BY-ND 4.0 | facts only |
| OSCP schemas (oscp/schemas) | schemas.zip 2020-09-18, sha256 72172497575e07db1dfb28c894f663a5d94b386d3718ffd31881940ee4ee4b48 | CC BY-ND 4.0 | the files, unmodified |
| OICP check (oicpspec/) | Hubject's OpenAPI files, fetched at test time at commit f5a8e2f | CC BY-SA 4.0 | nothing kept in the repository |
| Interop peers (interop/peer, interop/oscppeer) | extrawest_ocpi 2025.7.16 and pyoscp 0.1.0 from PyPI, run in containers; the app.py scripts are our own | MIT | nothing copied; read for behaviour only |

## Other implementations

extrawest_ocpi and pyoscp (both MIT) were read to write the interop peer scripts (how their
stores, authenticators and registration managers are called) and run as peers; no code of them is
in this module. No other implementation is recorded as having been read
for this code, and no similarity check against them was done.

## Tests

| Test | Source |
|---|---|
| ocpi/, oicp/, oscp/ unit tests and examples | our own cases from the documents above; demo values |
| oicpspec/ | Hubject's OpenAPI description at a pinned commit |
| interop/ | extrawest_ocpi 2025.7.16 and pyoscp 0.1.0 in Docker |
