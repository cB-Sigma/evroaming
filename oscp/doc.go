// Package oscp is the Open Smart Charging Protocol 2.0 of the Open Charge
// Alliance (FINAL, 2020-10-12): the messages of clause 4 and datatypes of
// clause 5, which role sends which message to which (CanSend), validation
// of every message against the OCA JSON schemas (embedded unmodified, with
// load-time corrections where a schema contradicts the text) and against
// rules the schemas lack, the HTTP headers, tokens and request IDs of
// 4.1.2, message segmentation (4.1.2.1) and a client that posts a message
// and reads the answer (4.1.3). Registration, handshake and heartbeat
// state, and what each role does with a message, are the application's.
package oscp
