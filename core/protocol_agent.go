package core

import (
	"github.com/geistesk/dtn7/cla"
)

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type ProtocolAgent struct {
	ApplicationAgent  *ApplicationAgent
	ConvergenceLayers []cla.ConvergenceLayer
}
