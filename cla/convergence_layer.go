// Package cla defines two interfaces for convergence layers.
//
// The ConvergenceReceiver specifies a type which receives bundles and forwards
// those to an exposed channel.
//
// The ConvergenceSender specifies a type which sends bundles to a remote
// endpoint.
//
// An implemented convergence layer can be a ConvergenceReceiver,
// ConvergenceSender or even both. This depends on the convergence layer's
// specification and is an implemention matter.
package cla

import "github.com/geistesk/dtn7/bundle"

// RecBundle is a tuple struct to attach the receiving CLA's node ID  to an
// incomming bundle. Each ConvergenceReceiver returns its received bundles as
// a channel of RecBundles.
type RecBundle struct {
	Bundle   bundle.Bundle
	Receiver bundle.EndpointID
}

// NewRecBundle returns a new RecBundle for the given bundle and CLA.
func NewRecBundle(b bundle.Bundle, rec bundle.EndpointID) RecBundle {
	return RecBundle{
		Bundle:   b,
		Receiver: rec,
	}
}

// ConvergenceReceiver is an interface for types which are able to receive
// bundles and write them to a channel. This channel can be accessed through
// the Channel method.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type ConvergenceReceiver interface {
	// Channel returns a channel of received bundles.
	Channel() <-chan RecBundle

	// Close signals this ConvergenceReceiver to shut down.
	Close()

	// GetEndpointID returns the endpoint ID assigned to this CLA.
	GetEndpointID() bundle.EndpointID
}

// ConvergenceSender is an interface for types which are able to transmit
// bundles to another node.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type ConvergenceSender interface {
	// Send transmits a bundle to this ConvergenceSender's endpoint. This method
	// should be thread safe and finish transmitting one bundle, before acting
	// on the next. This could be achieved by using a mutex or the like.
	Send(bndl bundle.Bundle) error

	// Close signals this ConvergenceSender to shut down.
	Close()

	// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
	// if it's known. Otherwise the zero endpoint will be returned.
	GetPeerEndpointID() bundle.EndpointID
}
