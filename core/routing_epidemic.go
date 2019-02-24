package core

import "github.com/geistesk/dtn7/cla"

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic routing way.
type EpidemicRouting struct {
	c *Core
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core.
func NewEpidemicRouting(c *Core) EpidemicRouting {
	return EpidemicRouting{c}
}

func (er EpidemicRouting) NotifyIncomming(_ BundlePack) {}

func (er EpidemicRouting) SenderForBundle(bp BundlePack) []cla.ConvergenceSender {
	return er.c.convergenceSenders
}
