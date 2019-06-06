package core

import "github.com/dtn7/dtn7-go/bundle"

// ApplicationAgent is an interface which describes application agent, which can
// both receive and transmit bundles in interaction with the Core.
type ApplicationAgent interface {
	// EndpointID returns this ApplicationAgent's (unique) endpoint ID.
	EndpointID() bundle.EndpointID

	// Deliver delivers a received bundle to this ApplicationAgent. This bundle
	// may contain an application specific payload or an administrative record.
	Deliver(bp BundlePack) error
}
