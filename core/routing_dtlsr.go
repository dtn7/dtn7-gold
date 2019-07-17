package core

import (
	"github.com/dtn7/dtn7-go/bundle"
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
	}
}
