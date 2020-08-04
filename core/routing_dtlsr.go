// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/RyanCarrier/dijkstra"
	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

const dtlsrBroadcastAddress = "dtn://routing/dtlsr/broadcast/"

type DTLSRConfig struct {
	// RecomputeTime is the interval (in seconds) until the routing table is recomputed.
	// Note: Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	RecomputeTime string
	// BroadcastTime is the interval (in seconds) between broadcasts of peer data.
	// Note: Broadcast only happens when there was a change in peer data.
	// Note: Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	BroadcastTime string
	// PurgeTime is the interval after which a disconnected peer is removed from the peer list.
	// Note: Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	PurgeTime string
}

// DTLSR is an implementation of "Delay Tolerant Link State Routing"
type DTLSR struct {
	c *Core
	// routingTable is a [endpoint]forwardingNode mapping
	routingTable map[bundle.EndpointID]bundle.EndpointID
	// peerChange denotes whether there has been a change in our direct connections
	// since we last calculated our routing table/broadcast our peer data
	peerChange bool
	// peers is our own peerData
	peers peerData
	// receivedChange denotes whether we received new data since we last computed our routing table
	receivedChange bool
	// receivedData is peerData received from other nodes
	receivedData map[bundle.EndpointID]peerData
	// nodeIndex and index Node are a bidirectional mapping EndpointID <-> uint64
	// necessary since the dijkstra implementation only accepts integer node identifiers
	nodeIndex map[bundle.EndpointID]int
	indexNode []bundle.EndpointID
	length    int
	// broadcastAddress is where metadata-bundles are sent to
	broadcastAddress bundle.EndpointID
	// purgeTime is the time until a peer gets removed from the peer list
	purgeTime time.Duration
	// dataMutex is a RW-mutex which protects change operations to the algorithm's metadata
	dataMutex sync.RWMutex
}

// peerData contains a peer's connection data
type peerData struct {
	// id is the node's endpoint id
	id bundle.EndpointID
	// timestamp is the time the last change occurred
	// when receiving other node's data, we only update if the timestamp in newer
	timestamp bundle.DtnTime
	// peers is a mapping of previously seen peers and the respective timestamp of the last encounter
	peers map[bundle.EndpointID]bundle.DtnTime
}

func (pd peerData) isNewerThan(other peerData) bool {
	return pd.timestamp > other.timestamp
}

func NewDTLSR(c *Core, config DTLSRConfig) *DTLSR {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("Initialising DTLSR")

	bAddress, err := bundle.NewEndpointID(dtlsrBroadcastAddress)
	if err != nil {
		log.WithFields(log.Fields{
			"dtlsrBroadcastAddress": dtlsrBroadcastAddress,
		}).Fatal("Unable to parse broadcast address")
	}

	purgeTime, err := time.ParseDuration(config.PurgeTime)
	if err != nil {
		log.WithFields(log.Fields{
			"string": config.PurgeTime,
		}).Fatal("Unable to parse duration")
	}

	dtlsr := DTLSR{
		c:            c,
		routingTable: make(map[bundle.EndpointID]bundle.EndpointID),
		peerChange:   false,
		peers: peerData{
			id:        c.NodeId,
			timestamp: bundle.DtnTimeNow(),
			peers:     make(map[bundle.EndpointID]bundle.DtnTime),
		},
		receivedChange:   false,
		receivedData:     make(map[bundle.EndpointID]peerData),
		nodeIndex:        map[bundle.EndpointID]int{c.NodeId: 0},
		indexNode:        []bundle.EndpointID{c.NodeId},
		length:           1,
		broadcastAddress: bAddress,
		purgeTime:        purgeTime,
	}

	err = c.cron.Register("dtlsr_purge", dtlsr.purgePeers, purgeTime)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Could not register DTLSR purge job")
	}

	recomputeTime, err := time.ParseDuration(config.RecomputeTime)
	if err != nil {
		log.WithFields(log.Fields{
			"string": config.RecomputeTime,
		}).Fatal("Unable to parse duration")
	}

	err = c.cron.Register("dtlsr_recompute", dtlsr.recomputeCron, recomputeTime)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Could not register DTLSR recompute job")
	}

	broadcastTime, err := time.ParseDuration(config.BroadcastTime)
	if err != nil {
		log.WithFields(log.Fields{
			"string": config.BroadcastTime,
		}).Fatal("Unable to parse duration")
	}

	err = c.cron.Register("dtlsr_broadcast", dtlsr.broadcastCron, broadcastTime)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Could not register DTLSR broadcast job")
	}

	// register our custom metadata-block
	extensionBlockManager := bundle.GetExtensionBlockManager()
	if !extensionBlockManager.IsKnown(bundle.ExtBlockTypeDTLSRBlock) {
		// since we already checked if the block type exists, this really shouldn't ever fail...
		_ = extensionBlockManager.Register(NewDTLSRBlock(dtlsr.peers))
	}

	return &dtlsr
}

func (dtlsr *DTLSR) NotifyIncoming(bp BundlePack) {
	if metaDataBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypeDTLSRBlock); err == nil {
		log.WithFields(log.Fields{
			"peer": bp.MustBundle().PrimaryBlock.SourceNode,
		}).Debug("Received metadata")

		dtlsrBlock := metaDataBlock.Value.(*DTLSRBlock)
		data := dtlsrBlock.getPeerData()

		log.WithFields(log.Fields{
			"peer": bp.MustBundle().PrimaryBlock.SourceNode,
			"data": data,
		}).Debug("Decoded peer data")

		dtlsr.dataMutex.Lock()
		defer dtlsr.dataMutex.Unlock()
		storedData, present := dtlsr.receivedData[data.id]

		if !present {
			log.Debug("Data for new peer")
			// if we didn't have any data for that peer, we simply add it
			dtlsr.receivedData[data.id] = data
			dtlsr.receivedChange = true

			// track node
			dtlsr.newNode(data.id)

			// track peers of this node
			for node := range data.peers {
				dtlsr.newNode(node)
			}
		} else {
			// check if the received data is newer and replace it if it is
			if data.isNewerThan(storedData) {
				log.Debug("Updating peer data")
				dtlsr.receivedData[data.id] = data
				dtlsr.receivedChange = true

				// track peers of this node
				for node := range data.peers {
					dtlsr.newNode(node)
				}
			}
		}
	}

	// store cla from which we received this bundle so that we don't always bounce bundles between nodes
	bundleItem, err := dtlsr.c.store.QueryId(bp.Id)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Debug("Bundle not in store")
		return
	}

	bndl := bp.MustBundle()

	if pnBlock, err := bndl.ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
		prevNode := pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint()

		sentEids, ok := bundleItem.Properties["routing/dtlsr/sent"].([]bundle.EndpointID)
		if !ok {
			sentEids = make([]bundle.EndpointID, 0)
		}

		bundleItem.Properties["routing/dtlsr/sent"] = append(sentEids, prevNode)
		if err := dtlsr.c.store.Update(bundleItem); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Updating BundleItem failed")
		}
	}
}

func (_ *DTLSR) ReportFailure(_ BundlePack, _ cla.ConvergenceSender) {
	// if the transmission failed, that is sad, but there is really nothing to do...
}

func (dtlsr *DTLSR) SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool) {
	delete = false

	bndl, err := bp.Bundle()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("Bundle no longer exists")
		return
	}

	if bndl.PrimaryBlock.Destination == dtlsr.broadcastAddress {
		bundleItem, err := dtlsr.c.store.QueryId(bp.Id)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Debug("Bundle not in store")
			return
		}

		sender, sentEids := filterCLAs(bundleItem, dtlsr.c.claManager.Sender(), "dtlsr")

		// broadcast bundles are always forwarded to everyone
		log.WithFields(log.Fields{
			"bundle":    bndl.ID(),
			"recipient": bndl.PrimaryBlock.Destination,
			"CLAs":      sender,
		}).Debug("Relaying broadcast bundle")

		bundleItem.Properties["routing/dtlsr/sent"] = sentEids
		if err := dtlsr.c.store.Update(bundleItem); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Updating BundleItem failed")
		}

		log.WithFields(log.Fields{
			"bundle": bndl.ID(),
			"peers":  sender,
		}).Debug("Forwarding metadata-bundle to theses peers.")
		return sender, delete
	}

	recipient := bndl.PrimaryBlock.Destination

	dtlsr.dataMutex.RLock()
	forwarder, present := dtlsr.routingTable[recipient]
	dtlsr.dataMutex.RUnlock()
	if !present {
		// we don't know where to forward this bundle
		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"recipient": recipient,
		}).Debug("DTLSR could not find a node to forward to")
		return
	}

	for _, cs := range dtlsr.c.claManager.Sender() {
		if cs.GetPeerEndpointID() == forwarder {
			sender = append(sender, cs)
			log.WithFields(log.Fields{
				"bundle":             bndl.ID(),
				"recipient":          recipient,
				"convergence-sender": sender,
			}).Debug("DTLSR selected Convergence Sender for an outgoing bundle")
			// we only ever forward to a single node
			// since DTLSR has no multiplicity for bundles
			// (we only ever forward it to the next node according to our routing table),
			// we can delete the bundle from our store after successfully forwarding it
			delete = true
			return
		}
	}

	log.WithFields(log.Fields{
		"bundle":    bp.ID(),
		"recipient": recipient,
	}).Debug("DTLSR could not find forwarder amongst connected nodes")
	return
}

func (dtlsr *DTLSR) ReportPeerAppeared(peer cla.Convergence) {
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

	dtlsr.dataMutex.Lock()
	defer dtlsr.dataMutex.Unlock()
	// track node
	dtlsr.newNode(peerID)

	// add node to peer list
	dtlsr.peers.peers[peerID] = 0
	dtlsr.peers.timestamp = bundle.DtnTimeNow()
	dtlsr.peerChange = true

	log.WithFields(log.Fields{
		"peer": peerID,
	}).Debug("Peer is now being tracked")
}

func (dtlsr *DTLSR) ReportPeerDisappeared(peer cla.Convergence) {
	log.WithFields(log.Fields{
		"address": peer,
	}).Debug("Peer disappeared")

	peerReceiver, ok := peer.(cla.ConvergenceSender)
	if !ok {
		log.Warn("Peer was not a ConvergenceSender")
		return
	}

	peerID := peerReceiver.GetPeerEndpointID()

	log.WithFields(log.Fields{
		"peer": peerID,
	}).Debug("PeerID discovered")

	dtlsr.dataMutex.Lock()
	defer dtlsr.dataMutex.Unlock()
	// set expiration timestamp for peer
	timestamp := bundle.DtnTimeNow()
	dtlsr.peers.peers[peerID] = timestamp
	dtlsr.peers.timestamp = timestamp
	dtlsr.peerChange = true

	log.WithFields(log.Fields{
		"peer": peer,
	}).Debug("Peer timeout is now running")
}

// DispatchingAllowed allows the processing of all packages.
func (_ *DTLSR) DispatchingAllowed(_ BundlePack) bool {
	// TODO: for future optimisation, we might track the timestamp of the last recomputation of the routing table
	// and only dispatch if it changed since the last time we tried.
	return true
}

// newNode adds a node to the index-mapping (if it was not previously tracked)
func (dtlsr *DTLSR) newNode(id bundle.EndpointID) {
	log.WithFields(log.Fields{
		"NodeID": id,
	}).Debug("Tracking Node")
	_, present := dtlsr.nodeIndex[id]

	if present {
		log.WithFields(log.Fields{
			"NodeID": id,
		}).Debug("Node already tracked")
		// node is already tracked
		return
	}

	dtlsr.nodeIndex[id] = dtlsr.length
	dtlsr.indexNode = append(dtlsr.indexNode, id)
	dtlsr.length = dtlsr.length + 1
	log.WithFields(log.Fields{
		"NodeID": id,
	}).Debug("Added node to tracking store")
}

// computeRoutingTable finds shortest paths using dijkstra's algorithm
func (dtlsr *DTLSR) computeRoutingTable() {
	log.Debug("Recomputing routing table")

	currentTime := bundle.DtnTimeNow()
	graph := dijkstra.NewGraph()

	// add vertices
	for i := 0; i < dtlsr.length; i++ {
		graph.AddVertex(i)
		// log node-index mapping for debug purposes
		log.WithFields(log.Fields{
			"index": i,
			"node":  dtlsr.indexNode[i],
		}).Debug("Node-index-mapping")
	}

	// add edges originating from this node
	for peer, timestamp := range dtlsr.peers.peers {
		var edgeCost int64
		if timestamp == 0 {
			edgeCost = 0
		} else {
			edgeCost = int64(currentTime - timestamp)
		}

		if err := graph.AddArc(0, dtlsr.nodeIndex[peer], edgeCost); err != nil {
			log.WithFields(log.Fields{
				"reason": err.Error(),
			}).Warn("Error computing routing table")
			return
		}

		log.WithFields(log.Fields{
			"peerA": dtlsr.c.NodeId,
			"peerB": peer,
			"cost":  edgeCost,
		}).Debug("Added vertex")
	}

	// add edges originating from other nodes
	for _, data := range dtlsr.receivedData {
		for peer, timestamp := range data.peers {
			var edgeCost int64
			if timestamp == 0 {
				edgeCost = 0
			} else {
				edgeCost = int64(currentTime - timestamp)
			}

			if err := graph.AddArc(dtlsr.nodeIndex[data.id], dtlsr.nodeIndex[peer], edgeCost); err != nil {
				log.WithFields(log.Fields{
					"reason": err.Error(),
				}).Warn("Error computing routing table")
				return
			}

			log.WithFields(log.Fields{
				"peerA": data.id,
				"peerB": peer,
				"cost":  edgeCost,
			}).Debug("Added vertex")
		}
	}

	routingTable := make(map[bundle.EndpointID]bundle.EndpointID)
	for i := 1; i < dtlsr.length; i++ {
		shortest, err := graph.Shortest(0, i)
		if err == nil {
			if len(shortest.Path) <= 1 {
				log.WithFields(log.Fields{
					"node_index": i,
					"node":       dtlsr.indexNode[i],
					"path":       shortest.Path,
				}).Warn("Single step path found - this should not happen")
				continue
			}

			routingTable[dtlsr.indexNode[i]] = dtlsr.indexNode[shortest.Path[1]]
			log.WithFields(log.Fields{
				"node_index": i,
				"node":       dtlsr.indexNode[i],
				"path":       shortest.Path,
				"next_hop":   routingTable[dtlsr.indexNode[i]],
			}).Debug("Found path to node")
		} else {
			log.WithFields(log.Fields{
				"node_index": i,
				"error":      err.Error(),
			}).Debug("Did not find path to node")
		}
	}

	log.WithFields(log.Fields{
		"routingTable": routingTable,
	}).Debug("Finished routing table computation")

	dtlsr.routingTable = routingTable
}

// recomputeCron gets called periodically by the core's cron module.
// Only actually triggers a recompute if the underlying data has changed.
func (dtlsr *DTLSR) recomputeCron() {
	dtlsr.dataMutex.RLock()
	peerChange := dtlsr.peerChange
	receivedChange := dtlsr.receivedChange
	dtlsr.dataMutex.RUnlock()

	log.WithFields(log.Fields{
		"peerChange":     peerChange,
		"receivedChange": receivedChange,
	}).Debug("Executing recomputeCron")

	if peerChange || receivedChange {
		dtlsr.dataMutex.Lock()
		dtlsr.computeRoutingTable()
		dtlsr.receivedChange = false
		dtlsr.dataMutex.Unlock()
	}
}

// broadcast broadcasts this node's peer data to the network
func (dtlsr *DTLSR) broadcast() {
	log.Debug("Broadcasting metadata")

	dtlsr.dataMutex.RLock()
	source := dtlsr.c.NodeId
	destination := dtlsr.broadcastAddress
	metadataBlock := NewDTLSRBlock(dtlsr.peers)
	dtlsr.dataMutex.RUnlock()

	err := sendMetadataBundle(dtlsr.c, source, destination, metadataBlock)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err.Error(),
		}).Warn("Unable to send metadata")
	}
}

// broadcastCron gets called periodically by the core's cron module.
// Only actually triggers a broadcast if peer data has changed
func (dtlsr *DTLSR) broadcastCron() {
	dtlsr.dataMutex.RLock()
	peerChange := dtlsr.peerChange
	dtlsr.dataMutex.RUnlock()

	log.WithFields(log.Fields{
		"peerChange": peerChange,
	}).Debug("Executing broadcastCron")

	if peerChange {
		dtlsr.broadcast()

		dtlsr.dataMutex.Lock()
		dtlsr.peerChange = false
		// a change in our own peer data should also trigger a routing recompute
		// but if this method gets called before recomputeCron(),
		// we don't want this information to be lost
		dtlsr.receivedChange = true
		dtlsr.dataMutex.Unlock()
	}
}

// purgePeers removes peers who have not been seen for a long time
func (dtlsr *DTLSR) purgePeers() {
	log.Debug("Executing purgePeers")
	currentTime := time.Now()

	dtlsr.dataMutex.Lock()
	defer dtlsr.dataMutex.Unlock()

	for peerID, timestamp := range dtlsr.peers.peers {
		if timestamp != 0 && timestamp.Time().Add(dtlsr.purgeTime).Before(currentTime) {
			log.WithFields(log.Fields{
				"peer":            peerID,
				"disconnect_time": timestamp,
			}).Debug("Removing stale peer")
			delete(dtlsr.peers.peers, peerID)
			dtlsr.peerChange = true
		}
	}
}

// DTLSRBlock contains routing metadata
//
// TODO: Turn this into an administrative record
type DTLSRBlock peerData

func NewDTLSRBlock(data peerData) *DTLSRBlock {
	newBlock := DTLSRBlock(data)
	return &newBlock
}

func (dtlsrb *DTLSRBlock) getPeerData() peerData {
	return peerData(*dtlsrb)
}

func (dtlsrb *DTLSRBlock) BlockTypeCode() uint64 {
	return bundle.ExtBlockTypeDTLSRBlock
}

func (dtlsrb *DTLSRBlock) CheckValid() error {
	return nil
}

func (dtlsrb *DTLSRBlock) MarshalCbor(w io.Writer) error {
	// start with the (apparently) required outer array
	if err := cboring.WriteArrayLength(3, w); err != nil {
		return err
	}

	// write our own endpoint id
	if err := cboring.Marshal(&dtlsrb.id, w); err != nil {
		return err
	}

	// write the timestamp
	if err := cboring.WriteUInt(uint64(dtlsrb.timestamp), w); err != nil {
		return err
	}

	// write the peer data array header
	if err := cboring.WriteMapPairLength(uint64(len(dtlsrb.peers)), w); err != nil {
		return err
	}

	// write the actual data
	for peerID, timestamp := range dtlsrb.peers {
		if err := cboring.Marshal(&peerID, w); err != nil {
			return err
		}
		if err := cboring.WriteUInt(uint64(timestamp), w); err != nil {
			return err
		}
	}

	return nil
}

func (dtlsrb *DTLSRBlock) UnmarshalCbor(r io.Reader) error {
	// read the (apparently) required outer array
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 3 {
		return fmt.Errorf("expected 3 fields, got %d", l)
	}

	// read endpoint id
	id := bundle.EndpointID{}
	if err := cboring.Unmarshal(&id, r); err != nil {
		return err
	} else {
		dtlsrb.id = id
	}

	// read the timestamp
	if timestamp, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		dtlsrb.timestamp = bundle.DtnTime(timestamp)
	}

	var lenData uint64

	// read length of data array
	lenData, err := cboring.ReadMapPairLength(r)
	if err != nil {
		return err
	}

	// read the actual data
	peers := make(map[bundle.EndpointID]bundle.DtnTime)
	var i uint64
	for i = 0; i < lenData; i++ {
		peerID := bundle.EndpointID{}
		if err := cboring.Unmarshal(&peerID, r); err != nil {
			return err
		}

		timestamp, err := cboring.ReadUInt(r)
		if err != nil {
			return err
		}

		peers[peerID] = bundle.DtnTime(timestamp)
	}

	dtlsrb.peers = peers

	return nil
}
