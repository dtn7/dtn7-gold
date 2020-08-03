// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cla defines two interfaces for Convergence Layer Adapters.
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
//
// Furthermore, the ConvergenceProvider provides the ability to create new
// instances of Convergence objects.
//
// Those types are generalized by the Convergable interface.
//
// A centralized instance for CLA management offers the Manager, designed to
// work seamlessly with the types above.
package cla

import "github.com/dtn7/dtn7-go/bundle"

// Convergable describes any kind of type which supports convergence layer-
// related services. This can be both a more specified Convergence interface
// type or a ConvergenceProvider.
type Convergable interface {
	// Close signals this Convergable to shut down.
	Close()
}

// Convergence is an interface to describe all kinds of Convergence Layer
// Adapters. There should not be a direct implemention of this interface. One
// must implement ConvergenceReceiver and/or ConvergenceSender, which are both
// extending this interface.
// A type can be both a ConvergenceReceiver and ConvergenceSender.
type Convergence interface {
	Convergable

	// Start starts this Convergence{Receiver,Sender} and might return an error
	// and a boolean indicating if another Start should be tried later.
	Start() (error, bool)

	// Channel represents a return channel for transmitted bundles, status
	// messages, etc.
	Channel() chan ConvergenceStatus

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

// ConvergenceProvider is a more general kind of CLA service which does not
// transfer any Bundles by itself, but supplies/creates new Convergence types.
// Those Convergence objects will be passed to a Manager. Thus, one might think
// of a ConvergenceProvider as some kind of middleware.
type ConvergenceProvider interface {
	Convergable

	// RegisterManager tells the ConvergenceProvider where to report new instances
	// of Convergence to.
	RegisterManager(*Manager)

	// Start starts this ConvergenceProvider. Before being started, the the
	// RegisterManager method tells this ConvergenceProvider its Manager. However,
	// the Manager will both call the RegisterManager and Start methods.
	Start() error
}
