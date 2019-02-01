// Package stcp provides a library for the Simple TCP Convergence-Layer Protocol
// as defined in draft-burleigh-dtn-stcp-00.txt.
//
// Because of the unidirectional design of STCP, both STPCServer and STCPClient
// exists. The STPCServer implements the ConvergenceReceiver and the STCPClient
// the ConvergenceSender interfaces defined in the parent cla package.
package stcp
