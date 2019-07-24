package core

import (
	"fmt"
	"github.com/RyanCarrier/dijkstra"
	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

// timestampNow outputs the current UNIX-time as an unsigned int64
// (I have no idea, why this is signed by default... does the kernel even allow you to set a negative time?)
func timestampNow() uint64 {
	return uint64(time.Now().Unix())
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
}

// peerData contains a peer's connection data
type peerData struct {
	// id is the node's endpoint id
	id bundle.EndpointID
	// timestamp is the time the last change occured
	// when receiving other node's data, we only update if the timestamp in newer
	timestamp uint64
	// peers is a mapping of previously seen peers and the respective timestamp of the last encounter
	peers map[bundle.EndpointID]uint64
}

func (pd peerData) isNewerThan(other peerData) bool {
	return pd.timestamp > other.timestamp
}

func NewDTLSR(c *Core) DTLSR {
	log.Debug("Initialised DTLSR")
	return DTLSR{
		c:            c,
		routingTable: make(map[bundle.EndpointID]bundle.EndpointID),
		peerChange:   false,
		peers: peerData{
			id:        c.NodeId,
			timestamp: timestampNow(),
			peers:     make(map[bundle.EndpointID]uint64),
		},
		receivedChange: false,
		receivedData:   make(map[bundle.EndpointID]peerData),
		nodeIndex:      map[bundle.EndpointID]int{c.NodeId: 0},
		indexNode:      []bundle.EndpointID{c.NodeId},
		length:         1,
	}
}

func (dtlsr DTLSR) NotifyIncoming(bp BundlePack) {
	if metaDataBlock, err := bp.Bundle.ExtensionBlock(ExtBlockTypeDTLSRBlock); err == nil {
		dtlsrBlock := metaDataBlock.Value.(*DTLSRBlock)
		data := dtlsrBlock.getPeerData()

		storedData, present := dtlsr.receivedData[data.id]

		if !present {
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
				dtlsr.receivedData[data.id] = data
				dtlsr.receivedChange = true

				// track peers of this node
				for node := range data.peers {
					dtlsr.newNode(node)
				}
			}
		}
	}
}

func (dtlsr DTLSR) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	// if the transmission failed, that is sad, but we don't really care...
	return
}

func (dtlsr DTLSR) SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool) {
	delete = false

	// TODO: handle broadcasts?

	recipient := bp.Bundle.PrimaryBlock.Destination
	forwarder, present := dtlsr.routingTable[recipient]
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
				"bundle":              bp.ID(),
				"recipient":           recipient,
				"convergence-senders": sender,
			}).Debug("DTLSR selected Convergence Sender for an outgoing bundle")
			// we only ever forward to a single node
			return
		}
	}

	log.WithFields(log.Fields{
		"bundle":    bp.ID(),
		"recipient": recipient,
	}).Debug("DTLSR could not find forwarder amongst connected nodes")
	return
}

func (dtlsr DTLSR) ReportPeerAppeared(peer cla.Convergence) {
	// generate EndpointID from address-string
	peerID, err := bundle.NewEndpointID(peer.Address())
	if err != nil {
		log.WithFields(log.Fields{
			"address": peer.Address(),
		}).Warn("Peers address can't pe parsed")
		return
	}

	// track node
	dtlsr.newNode(peerID)

	// add node to peer list
	dtlsr.peers.peers[peerID] = 0
	dtlsr.peers.timestamp = timestampNow()
	dtlsr.peerChange = true
}

func (dtlsr DTLSR) ReportPeerDisappeared(peer cla.Convergence) {

}

// newNode adds a node to the index-mapping (if it was not previously tracked)
func (dtlsr DTLSR) newNode(id bundle.EndpointID) {
	_, present := dtlsr.nodeIndex[id]

	if present {
		// node is already tracked
		return
	}

	dtlsr.nodeIndex[id] = dtlsr.length
	dtlsr.indexNode = append(dtlsr.indexNode, id)
	dtlsr.length = dtlsr.length + 1
}

// computeRoutingTable finds shortest paths using dijkstra's algorithm
func (dtlsr DTLSR) computeRoutingTable() {
	currentTime := timestampNow()
	graph := dijkstra.NewGraph()

	// add vertices
	for i := 0; i < dtlsr.length; i++ {
		graph.AddVertex(i)
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
				"reason": err,
			}).Warn("Error computing routing table")
			return
		}
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
					"reason": err,
				}).Warn("Error computing routing table")
				return
			}
		}
	}

	routingTable := make(map[bundle.EndpointID]bundle.EndpointID)
	for i := 1; i < dtlsr.length; i++ {
		shortest, err := graph.Shortest(0, i)
		if err == nil {
			routingTable[dtlsr.indexNode[0]] = dtlsr.indexNode[shortest.Path[0]]
		}
	}

	dtlsr.routingTable = routingTable
}

const ExtBlockTypeDTLSRBlock uint64 = 193

// DTLSRBlock contains routing metadata
type DTLSRBlock peerData

func NewDTLSRBlock(data peerData) *DTLSRBlock {
	newBlock := DTLSRBlock(data)
	return &newBlock
}

func (dtlsrb *DTLSRBlock) getPeerData() peerData {
	return peerData(*dtlsrb)
}

func (dtlsrb *DTLSRBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeDTLSRBlock
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
	if err := cboring.WriteUInt(dtlsrb.timestamp, w); err != nil {
		return err
	}

	// write the peer data array header
	if err := cboring.WriteArrayLength(uint64(len(dtlsrb.peers)), w); err != nil {
		return err
	}

	// write the actual data
	for peerID, timestamp := range dtlsrb.peers {
		if err := cboring.Marshal(&peerID, w); err != nil {
			return err
		}
		if err := cboring.WriteUInt(timestamp, w); err != nil {
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
		return fmt.Errorf("expected 4 fields, got %d", l)
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
		dtlsrb.timestamp = timestamp
	}

	var lenData uint64

	// read length of data array
	lenData, err := cboring.ReadArrayLength(r)
	if err != nil {
		return err
	}

	// read the actual data
	peers := make(map[bundle.EndpointID]uint64)
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

		peers[peerID] = timestamp
	}

	dtlsrb.peers = peers

	return nil
}
