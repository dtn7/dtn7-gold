package core

import (
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)
import log "github.com/sirupsen/logrus"

const PINIT = 0.75
const BETA = 0.25
const GAMMA = 0.98

type Prophet struct {
	c *Core
	// Mapping NodeID->Encounter Probability
	predictability map[bundle.EndpointID]float64
	// Map containing the predictability-maps of other nodes
	peerPredictabilities map[bundle.EndpointID]map[bundle.EndpointID]float64
}

func NewProphet(c *Core) *Prophet {
	log.WithFields(log.Fields{
		"p_init": PINIT,
		"beta":   BETA,
		"gamma":  GAMMA,
	}).Info("Initialised Prophet")

	prophet := Prophet{
		c:                    c,
		predictability:       make(map[bundle.EndpointID]float64),
		peerPredictabilities: make(map[bundle.EndpointID]map[bundle.EndpointID]float64),
	}

	return &prophet
}

// encounter updates the predictability for an encountered node
func encounter(prophet *Prophet, peer bundle.EndpointID) {
	pOld := prophet.predictability[peer]
	pNew := pOld + ((1 - pOld) * PINIT)
	prophet.predictability[peer] = pNew
	log.WithFields(log.Fields{
		"peer": peer,
		"pred": pNew,
	}).Debug("Updated predictability via encounter")
}

// agePred "ages" - decreases over time - the predictability for a node
func agePred(prophet *Prophet, peer bundle.EndpointID) {
	pOld := prophet.predictability[peer]
	pNew := pOld * GAMMA
	prophet.predictability[peer] = pNew
	log.WithFields(log.Fields{
		"peer": peer,
		"pred": pNew,
	}).Debug("Updated predictability via ageing")
}

// transitivity
func transitivity(prophet *Prophet, peer bundle.EndpointID) {
	peerPredictabilities, present := prophet.peerPredictabilities[peer]
	if !present {
		log.WithFields(log.Fields{
			"peer": peer,
		}).Debug("Don't know peer's predictabilities")
		return
	}

	log.WithFields(log.Fields{
		"peer": peer,
	}).Debug("Updating transitive predictabilities")

	for otherPeer, otherPeerPred := range peerPredictabilities {
		peerPred := prophet.predictability[peer]
		pOld := prophet.predictability[otherPeer]
		pNew := pOld + ((1 - pOld) * peerPred * otherPeerPred * BETA)
		prophet.predictability[otherPeer] = pNew
		log.WithFields(log.Fields{
			"peer":       peer,
			"other_peer": otherPeer,
			"pred":       pNew,
		}).Debug("Updated predictability via transitivity")
	}
}

// TODO; dummy implementation
func (prophet *Prophet) NotifyIncoming(bp BundlePack) {

}

// TODO: dummy implementation
func (prophet *Prophet) DispatchingAllowed(bp BundlePack) bool {
	return true
}

// TODO: dummy implementation
func (prophet *Prophet) SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool) {
	return nil, false
}

// TODO: dummy implementation
func (prophet *Prophet) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {

}

// TODO: dummy implementation
func (prophet *Prophet) ReportPeerAppeared(peer cla.Convergence) {

}

// TODO: dummy implementation
func (prophet *Prophet) ReportPeerDisappeared(peer cla.Convergence) {

}
