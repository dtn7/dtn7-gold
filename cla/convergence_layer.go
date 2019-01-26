package cla

import "github.com/geistesk/dtn7/bundle"

// ConvergenceReceiver is an interface for types which are able to receive
// bundles and write them to a channel. This channel can be accessed through
// the Channel method.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type ConvergenceReceiver interface {
	// Channel returns a channel of received bundles.
	Channel() <-chan bundle.Bundle

	// Close signals this ConvergenceReceiver to shut down.
	Close()

	// GetEndpointID returns the endpoint ID assigned to this CLA.
	GetEndpointID() bundle.EndpointID
}

// ConvergenceSender is an interface for types which are able to transmit
// bundles to another node.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type ConvergenceSender interface {
	// Send transmits a bundle to this ConvergenceSender's endpoint.
	Send(bndl bundle.Bundle) error

	// Close signals this ConvergenceSender to shut down.
	Close()

	// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
	// if it's known. Otherwise the zero endpoint will be returned.
	GetPeerEndpointID() bundle.EndpointID
}
