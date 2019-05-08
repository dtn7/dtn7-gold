package core

import "github.com/dtn7/dtn7/cla"

// RoutingAlgorithm is an interface to specify routing algorithms for
// delay-tolerant networks. An implementation might store a reference to a Core
// struct to refer the ConvergenceSenders.
type RoutingAlgorithm interface {
	// NotifyIncoming notifies this RoutingAlgorithm about incoming bundles.
	// Whether the algorithm acts on this information or ignores it, is both a
	// design and implementation decision.
	NotifyIncoming(bp BundlePack)

	// SenderForBundle returns an array of ConvergenceSender for a requested
	// bundle. Furthermore the finished flags indicates if this BundlePack should
	// be deleted afterwards.
	// The CLA selection is based on the algorithm's design.
	SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool)
}
