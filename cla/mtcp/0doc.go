// Package mtcp provides a library for the Minimal TCP Convergence-Layer
// Protocol as defined in draft-ietf-dtn-mtcpcl-01
//
// Because of the unidirectional design of MTCP, both MTPCServer and MTCPClient
// exists. The MTPCServer implements the ConvergenceReceiver and the MTCPClient
// the ConvergenceSender interfaces defined in the parent cla package.
package mtcp
