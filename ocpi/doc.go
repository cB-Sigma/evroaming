// Package ocpi is OCPI 2.2.1 (the Open Charge Point Interface of the
// EVRoaming Foundation): the objects of every module as they travel, the
// client side of a call (token encoding, routing headers, status codes,
// pagination) with a helper for every module call from either side, the
// server side (Router, which serves the Sender and Receiver interfaces an
// application implements, and the helpers it is built from: the response
// envelope, credentials token parsing, routing headers, paginated lists)
// and the credentials handshake steps. Partners that offer OCPI 2.2 are
// spoken to as 2.2.1. An application keeps its own partner state and
// passes its calls through the Doer it supplies.
package ocpi
