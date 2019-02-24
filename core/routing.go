package core

import "github.com/geistesk/dtn7/cla"

// RoutingAlgorithm is an interface to specify routing algorithms for
// delay-tolerant networks. An implementation might store a reference to a Core
// struct to refer the ConvergenceSenders.
type RoutingAlgorithm interface {
	// NotifyIncomming notifies this RoutingAlgorithm about incomming bundles.
	// Whether the algorithm acts on this information or ignores it, is both a
	// design and implementation decision.
	NotifyIncomming(bp BundlePack)

	// SenderForBundle returns an array of ConvergenceSender for a requested
	// bundle. The CLA selection is based on the algorithm's design.
	SenderForBundle(bp BundlePack) []cla.ConvergenceSender
}
