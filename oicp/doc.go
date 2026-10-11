// Package oicp is Hubject's Open InterCharge Protocol 2.3 for a charge
// point operator: the messages of every CPO operation as they travel
// (EVSE data and status, pricing products and EVSE pricing, authorise
// start and stop, charge detail records, charging notifications, remote
// start and stop, remote reservation start and stop), the identifier
// forms and status codes, the paths of the operations, a Client that
// posts the CPO's messages to Hubject and turns Hubject's answers into
// errors, and a Handler that serves the operations Hubject sends to the
// CPO. The CPO's own state, what a remote start does to a charger and
// the retry queue are the application's.
package oicp
