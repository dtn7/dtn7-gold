package core

import "github.com/geistesk/dtn7/cla"

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic routing way.
type EpidemicRouting struct {
	c        *Core
	sendBack bool
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core. The second parameter indicates if a bundle should also
// be send back to its origin.
func NewEpidemicRouting(c *Core, sendBack bool) EpidemicRouting {
	return EpidemicRouting{
		c:        c,
		sendBack: sendBack,
	}
}

// NotifyIncoming tells the EpidemicRouting new bundles. However,
// EpidemicRouting simply does not listen.
func (er EpidemicRouting) NotifyIncoming(_ BundlePack) {}

// SenderForBundle returns the Core's ConvergenceSenders. The ConvergenceSender
// for this BundlePack's receiver will be removed sendBack is false.
func (er EpidemicRouting) SenderForBundle(bp BundlePack) ([]cla.ConvergenceSender, bool) {
	var css []cla.ConvergenceSender
	for _, cs := range er.c.convergenceSenders {
		if !er.sendBack && cs.GetPeerEndpointID() != bp.Receiver {
			css = append(css, cs)
		}
	}

	return css, false
}
