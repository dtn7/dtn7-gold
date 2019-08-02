package core

import "github.com/dtn7/dtn7-go/cla"

// RoutingAlgorithm is an interface to specify routing algorithms for
// delay-tolerant networks. An implementation might store a reference to a Core
// struct to refer the ConvergenceSenders.
type RoutingAlgorithm interface {
	// NotifyIncoming notifies this RoutingAlgorithm about incoming bundles.
	// Whether the algorithm acts on this information or ignores it, is both a
	// design and implementation decision.
	NotifyIncoming(bp BundlePack)

	// DispatchingAllowed will be called from within the *dispatching* step of
	// the processing pipeline. A RoutingAlgorithm is allowed to drop the
	// proceeding of a bundle before being inspected further or being delivered
	// locally or to another node.
	DispatchingAllowed(bp BundlePack) bool

	// SenderForBundle returns an array of ConvergenceSender for a requested
	// bundle. Furthermore the finished flags indicates if this BundlePack should
	// be deleted afterwards.
	// The CLA selection is based on the algorithm's design.
	SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool)

	// ReportFailure notifies the RoutingAlgorithm about a failed transmission to
	// a previously selected CLA. Compare: SenderForBundle.
	ReportFailure(bp BundlePack, sender cla.ConvergenceSender)

	// ReportPeerAppeared notifies the RoutingAlgorithm about a new neighbor.
	ReportPeerAppeared(peer cla.Convergence)

	// ReportPeerDisappeared notifies the RoutingAlgorithm about the
	// disappearance of a neighbor.
	ReportPeerDisappeared(peer cla.Convergence)
}

// RoutingConfig contains necessary configuration data to initialise a routing algorithm
type RoutingConf struct {
	// Algorithm is one of the implemented routing-algorithms
	// May be: "epidemic", "spray", "binary_spray", "dtlsr"
	Algorithm string
	// DTLSRConf contains data to initialise dtlsr
	DTLSRConf DTLSRConfig
}
