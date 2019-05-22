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

import "github.com/dtn7/dtn7/bundle"

// RecBundle is a tuple struct to attach the receiving CLA's node ID  to an
// incoming bundle. Each ConvergenceReceiver returns its received bundles as
// a channel of RecBundles.
type RecBundle struct {
	Bundle   *bundle.Bundle
	Receiver bundle.EndpointID
}

// NewRecBundle returns a new RecBundle for the given bundle and CLA.
func NewRecBundle(b *bundle.Bundle, rec bundle.EndpointID) RecBundle {
	return RecBundle{
		Bundle:   b,
		Receiver: rec,
	}
}

// Convergence is an interface to describe all kinds of Convergence Layer
// Adapters. There should not be a direct implemention of this interface. One
// must implement ConvergenceReceiver and/or ConvergenceSender, which are both
// extending this interface.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type Convergence interface {
	// Start starts this Convergence{Receiver,Sender} and might return an error
	// and a boolean indicating if another Start should be tried later.
	Start() (error, bool)

	// Close signals this Convergence{Receiver,Send} to shut down.
	Close()

	// Address should return a unique address string to both identify this
	// Convergence{Receiver,Sender} and ensure it will not opened twice.
	Address() string

	// IsPermanent returns true, if this CLA should not be removed after failures.
	IsPermanent() bool
}

// ConvergenceReceiver is an interface for types which are able to receive
// bundles and write them to a channel. This channel can be accessed through
// the Channel method.
type ConvergenceReceiver interface {
	Convergence

	// Channel returns a channel of received bundles.
	Channel() chan RecBundle

	// GetEndpointID returns the endpoint ID assigned to this CLA.
	GetEndpointID() bundle.EndpointID
}

// ConvergenceSender is an interface for types which are able to transmit
// bundles to another node.
type ConvergenceSender interface {
	Convergence

	// Send transmits a bundle to this ConvergenceSender's endpoint. This method
	// should be thread safe and finish transmitting one bundle, before acting
	// on the next. This could be achieved by using a mutex or the like.
	Send(bndl *bundle.Bundle) error

	// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
	// if it's known. Otherwise the zero endpoint will be returned.
	GetPeerEndpointID() bundle.EndpointID
}
