package core

import (
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"sync"
	"time"
)
import log "github.com/sirupsen/logrus"

type ProphetConfig struct {
	// PInit ist the prophet initialisation constant
	PInit float64
	// Beta is the prophet scaling factor for transitive predictability
	Beta float64
	// Gamma is the prophet ageing factor
	Gamma float64
	// AgeInterval is the duration after which entries are aged
	AgeInterval string
}

type Prophet struct {
	c *Core
	// Mapping NodeID->Encounter Probability
	predictability map[bundle.EndpointID]float64
	// Map containing the predictability-maps of other nodes
	peerPredictabilities map[bundle.EndpointID]map[bundle.EndpointID]float64
	// dataMutex is a RW-mutex which protects change operations to the algorithm's metadata
	dataMutex sync.RWMutex
	// config contains the values for prophet constants
	config ProphetConfig
}

func NewProphet(c *Core, config ProphetConfig) *Prophet {
	log.WithFields(log.Fields{
		"p_init":       config.PInit,
		"beta":         config.Beta,
		"gamma":        config.Gamma,
		"age_interval": config.AgeInterval,
	}).Info("Initialised Prophet")

	prophet := Prophet{
		c:                    c,
		predictability:       make(map[bundle.EndpointID]float64),
		peerPredictabilities: make(map[bundle.EndpointID]map[bundle.EndpointID]float64),
		config:               config,
	}

	ageInterval, err := time.ParseDuration(config.AgeInterval)
	if err != nil {
		log.WithFields(log.Fields{
			"string": config.AgeInterval,
		}).Fatal("Unable to parse duration")
	}

	err = c.cron.Register("dtlsr_recompute", prophet.ageCron, ageInterval)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Could not register DTLSR recompute job")
	}

	return &prophet
}

// encounter updates the predictability for an encountered node
func (prophet *Prophet) encounter(peer bundle.EndpointID) {
	pOld := prophet.predictability[peer]
	pNew := pOld + ((1 - pOld) * prophet.config.PInit)
	prophet.predictability[peer] = pNew
	log.WithFields(log.Fields{
		"peer": peer,
		"pOld": pOld,
		"pNew": pNew,
	}).Debug("Updated predictability via encounter")
}

// agePred "ages" - decreases over time - the predictability for a node
func (prophet *Prophet) agePred(peer bundle.EndpointID) {
	pOld := prophet.predictability[peer]
	pNew := pOld * prophet.config.Gamma
	prophet.predictability[peer] = pNew
	log.WithFields(log.Fields{
		"peer": peer,
		"pOld": pOld,
		"pNew": pNew,
	}).Debug("Updated predictability via ageing")
}

// ageCron gets called periodically by the core's cron ange ages all peer predictabilities
func (prophet *Prophet) ageCron() {
	prophet.dataMutex.Lock()
	defer prophet.dataMutex.Unlock()
	for peer := range prophet.predictability {
		prophet.agePred(peer)
	}
}

// transitivity increases predicability for nodes based on a peer's corresponding predicability
// If we are likely to reencounter node b and node b is likely to reencounter node c
// then we are also a good forwarder for node c
func (prophet *Prophet) transitivity(peer bundle.EndpointID) {
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
		pNew := pOld + ((1 - pOld) * peerPred * otherPeerPred * prophet.config.Beta)
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

func (prophet *Prophet) ReportPeerAppeared(peer cla.Convergence) {
	log.WithFields(log.Fields{
		"address": peer,
	}).Debug("Peer appeared")

	peerReceiver, ok := peer.(cla.ConvergenceSender)
	if !ok {
		log.Warn("Peer was not a ConvergenceSender")
		return
	}

	peerID := peerReceiver.GetPeerEndpointID()

	log.WithFields(log.Fields{
		"peer": peerID,
	}).Debug("PeerID discovered")

	prophet.dataMutex.Lock()
	defer prophet.dataMutex.Unlock()

	prophet.encounter(peerID)
}

// TODO: dummy implementation
func (prophet *Prophet) ReportPeerDisappeared(peer cla.Convergence) {

}
