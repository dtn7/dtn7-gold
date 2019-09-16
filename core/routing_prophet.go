package core

import (
	"fmt"
	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"io"
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

// predictabilities contains a node's ID as well as the delivery predictabilities for other nodes
type predictabilities struct {
	// id is this node's EndpointID
	id bundle.EndpointID
	// Mapping NodeID->Encounter Probability
	predictability map[bundle.EndpointID]float64
}

type Prophet struct {
	c *Core
	// predictabilities are this node's delivery probabilities for other nodes
	predictabilities predictabilities
	// Map containing the predictability-maps of other nodes
	peerPredictabilities map[bundle.EndpointID]predictabilities
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
		c: c,
		predictabilities: predictabilities{
			id:             c.NodeId,
			predictability: make(map[bundle.EndpointID]float64),
		},
		peerPredictabilities: make(map[bundle.EndpointID]predictabilities),
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

	// register our custom metadata-block
	extensionBlockManager := bundle.GetExtensionBlockManager()
	if !extensionBlockManager.IsKnown(ExtBlockTypeProphetBlock) {
		// since we already checked if the block type exists, this really shouldn't ever fail...
		_ = extensionBlockManager.Register(newProphetBlock(prophet.predictabilities))
	}

	return &prophet
}

// encounter updates the predictability for an encountered node
func (prophet *Prophet) encounter(peer bundle.EndpointID) {
	pOld := prophet.predictabilities.predictability[peer]
	pNew := pOld + ((1 - pOld) * prophet.config.PInit)
	prophet.predictabilities.predictability[peer] = pNew
	log.WithFields(log.Fields{
		"peer": peer,
		"pOld": pOld,
		"pNew": pNew,
	}).Debug("Updated predictability via encounter")
}

// agePred "ages" - decreases over time - the predictability for a node
func (prophet *Prophet) agePred(peer bundle.EndpointID) {
	pOld := prophet.predictabilities.predictability[peer]
	pNew := pOld * prophet.config.Gamma
	prophet.predictabilities.predictability[peer] = pNew
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
	for peer := range prophet.predictabilities.predictability {
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

	for otherPeer, otherPeerPred := range peerPredictabilities.predictability {
		peerPred := prophet.predictabilities.predictability[peer]
		pOld := prophet.predictabilities.predictability[otherPeer]
		pNew := pOld + ((1 - pOld) * peerPred * otherPeerPred * prophet.config.Beta)
		prophet.predictabilities.predictability[otherPeer] = pNew
		log.WithFields(log.Fields{
			"beta":            prophet.config.Beta,
			"peer":            peer,
			"peer_pred":       peerPred,
			"other_peer":      otherPeer,
			"other_peer_pred": otherPeerPred,
			"pOld":            pOld,
			"pNew":            pNew,
		}).Debug("Updated predictability via transitivity")
	}
}

// sendMetadata sends our summary-vector with our delivery predictabilities to a peer
func (prophet *Prophet) sendMetadata(recipient bundle.EndpointID) {
	prophet.dataMutex.RLock()

	bundleBuilder := bundle.Builder()
	bundleBuilder.Destination(recipient)
	bundleBuilder.Source(prophet.c.NodeId)
	bundleBuilder.CreationTimestampNow()
	bundleBuilder.Lifetime("10m")
	bundleBuilder.BundleCtrlFlags(bundle.MustNotFragmented)
	// no Payload
	bundleBuilder.PayloadBlock(byte(1))

	metadataBlock := newProphetBlock(prophet.predictabilities)

	prophet.dataMutex.RUnlock()

	bundleBuilder.Canonical(metadataBlock)
	metadatBundle, err := bundleBuilder.Build()
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Unable to build metadata bundle")
		return
	} else {
		log.Debug("Metadata Bundle built")
	}

	log.Debug("Sending metadata bundle")
	prophet.c.SendBundle(&metadatBundle)
	log.WithFields(log.Fields{
		"bundle": metadatBundle,
	}).Debug("Successfully sent metadata bundle")
}

func (prophet *Prophet) NotifyIncoming(bp BundlePack) {
	if metaDataBlock, err := bp.MustBundle().ExtensionBlock(ExtBlockTypeProphetBlock); err == nil {
		log.WithFields(log.Fields{
			"source": bp.MustBundle().PrimaryBlock.SourceNode,
		}).Debug("Received metadata")

		if bp.MustBundle().PrimaryBlock.Destination != prophet.c.NodeId {
			log.WithFields(log.Fields{
				"recipient": bp.MustBundle().PrimaryBlock.Destination,
				"own_id":    prophet.c.NodeId,
			}).Debug("Received Metadata meant for different node")
			return
		}

		prophetBlock := metaDataBlock.Value.(*ProphetBlock)
		data := prophetBlock.getPredictabilities()

		log.WithFields(log.Fields{
			"source": bp.MustBundle().PrimaryBlock.SourceNode,
			"data":   data,
		}).Debug("Decoded peer data")

		prophet.dataMutex.Lock()
		defer prophet.dataMutex.Unlock()

		_, present := prophet.peerPredictabilities[data.id]
		if present {
			log.WithFields(log.Fields{
				"peer": data.id,
			}).Debug("Updating peer metadata")
		} else {
			log.WithFields(log.Fields{
				"peer": data.id,
			}).Debug("Metadata for new peer")
		}

		// import new metadata
		prophet.peerPredictabilities[data.id] = data

		// update own predictabilities via the transitive property
		prophet.transitivity(data.id)
	}
}

// TODO: dummy implementation
func (prophet *Prophet) DispatchingAllowed(bp BundlePack) bool {
	return true
}

// TODO: dummy implementation
func (prophet *Prophet) SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool) {
	return nil, false
}

func (prophet *Prophet) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	// When a transmission fails, that's unfortunate bet there really is not a whole lot to do
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

	// update our delivery predictability for this peer
	prophet.dataMutex.Lock()
	prophet.encounter(peerID)
	prophet.dataMutex.Unlock()

	// send them our summary vector
	prophet.sendMetadata(peerID)
}

// TODO: dummy implementation
func (prophet *Prophet) ReportPeerDisappeared(peer cla.Convergence) {

}

// TODO: Turn this into an administrative record

const ExtBlockTypeProphetBlock uint64 = 194

// DTLSRBlock contains routing metadata
type ProphetBlock predictabilities

func newProphetBlock(data predictabilities) *ProphetBlock {
	newBlock := ProphetBlock(data)
	return &newBlock
}

func (pBlock *ProphetBlock) getPredictabilities() predictabilities {
	return predictabilities(*pBlock)
}

func (pBlock *ProphetBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeProphetBlock
}

func (pBlock ProphetBlock) CheckValid() error {
	return nil
}

func (pBlock *ProphetBlock) MarshalCbor(w io.Writer) error {
	// start with the outer array
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	// write endpoint id
	if err := cboring.Marshal(&pBlock.id, w); err != nil {
		return err
	}

	// write the peer data array header
	if err := cboring.WriteArrayLength(uint64(len(pBlock.predictability)), w); err != nil {
		return err
	}

	// write the actual data
	for peerID, pred := range pBlock.predictability {
		if err := cboring.Marshal(&peerID, w); err != nil {
			return err
		}
		if err := cboring.WriteFloat64(pred, w); err != nil {
			return err
		}
	}

	return nil
}

func (pBlock *ProphetBlock) UnmarshalCbor(r io.Reader) error {
	// read the outer array
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("expected 2 fields, got %d", l)
	}

	// read endpoint id
	id := bundle.EndpointID{}
	if err := cboring.Unmarshal(&id, r); err != nil {
		return err
	} else {
		pBlock.id = id
	}

	var lenData uint64

	// read length of data array
	lenData, err := cboring.ReadArrayLength(r)
	if err != nil {
		return err
	}

	// read the actual data
	predictability := make(map[bundle.EndpointID]float64)
	var i uint64
	for i = 0; i < lenData; i++ {
		peerID := bundle.EndpointID{}
		if err := cboring.Unmarshal(&peerID, r); err != nil {
			return err
		}

		pred, err := cboring.ReadFloat64(r)
		if err != nil {
			return err
		}

		predictability[peerID] = pred
	}

	pBlock.predictability = predictability

	return nil
}
