package cla

import "github.com/geistesk/dtn7/bundle"

// ConvergenceLayer is an interface for convergence layer adapters (CLA). Each
// CLA should work in an own thread/Goroutine, which is started with the
// Construct and terminated with the Destruct method. New bundles can be
// transmitted to a known bundle node, identified by its endpoint ID, by calling
// the SendBundle method.
// TODO: reception, threading, ...
type ConvergenceLayer interface {
	// TODO: connection to BPA with an observer pattern or the like

	// GetEndpointID returns the endpoint ID assigned to this CLA.
	GetEndpointID() bundle.EndpointID

	// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer.
	GetPeerEndpointID() bundle.EndpointID

	// Construct setups this convergence layer adapter's Goroutine.
	Construct()

	// Destruct terminates this convergence layer adapter's Goroutine.
	Destruct()

	// SendBundle transmits the given bundle to a node.
	SendBundle(recipient bundle.EndpointID, bndl bundle.Bundle)
}
