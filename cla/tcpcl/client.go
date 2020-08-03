// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// sessTermErr will be returned from a state handler iff a SESS_TERM was received.
var sessTermErr = errors.New("SESS_TERM received")

// Client is a TCPCL client for a bidirectional Bundle exchange. Thus, the Client type implements both
// cla.ConvergenceReceiver and cla.ConvergenceSender. A Client can be created by the Listener for incoming
// connections or dialed for outbounding connections.
type Client struct {
	address        string
	started        bool
	permanent      bool
	endpointID     bundle.EndpointID
	peerEndpointID bundle.EndpointID

	conn net.Conn

	msgsOut chan Message
	msgsIn  chan Message

	handleMetaStop        chan struct{}
	handleMetaStopAck     chan struct{}
	handlerConnInStop     chan struct{}
	handlerConnInStopAck  chan struct{}
	handlerConnOutStop    chan struct{}
	handlerConnOutStopAck chan struct{}
	handlerStateStop      chan struct{}
	handlerStateStopAck   chan struct{}

	active bool
	state  *ClientState

	// Contact state fields:
	contactSent bool
	contactRecv bool
	chSent      ContactHeader
	chRecv      ContactHeader

	// Init state fields:
	initSent     bool
	initRecv     bool
	sessInitSent SessionInitMessage
	sessInitRecv SessionInitMessage

	keepalive   uint16
	segmentMru  uint64
	transferMru uint64

	// Established state fields:
	keepaliveStarted bool
	keepaliveLast    time.Time
	keepaliveTicker  *time.Ticker

	transferOutMutex sync.Mutex
	transferOutId    uint64
	transferOutSend  chan Message
	transferOutAck   chan Message

	transferIn *IncomingTransfer

	reportChan chan cla.ConvergenceStatus
}

// NewClient creates a new Client on an existing connection. This function is used from the Listener.
func NewClient(conn net.Conn, endpointID bundle.EndpointID) *Client {
	return &Client{
		address:    conn.RemoteAddr().String(),
		conn:       conn,
		active:     false,
		endpointID: endpointID,
	}
}

// DialClient tries to establish a new TCPCL Client to a remote server.
func DialClient(address string, endpointID bundle.EndpointID, permanent bool) *Client {
	return &Client{
		address:    address,
		permanent:  permanent,
		active:     true,
		endpointID: endpointID,
	}
}

func (client *Client) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TCPCL(")
	if client.conn != nil {
		fmt.Fprintf(&b, "peer=%v, ", client.conn.RemoteAddr())
	} else {
		fmt.Fprintf(&b, "peer=NONE, ")
	}
	fmt.Fprintf(&b, "active peer=%t", client.active)
	fmt.Fprintf(&b, ")")

	return b.String()
}

// log prepares a new log entry with predefined session data.
func (client *Client) log() *log.Entry {
	return log.WithFields(log.Fields{
		"session": client,
		"state":   client.state,
	})
}

func (client *Client) Start() (err error, retry bool) {
	client.state = new(ClientState)

	if client.started {
		if client.active {
			client.log().Debug("Clearing connection for reactivation")
			<-client.handleMetaStopAck
			client.conn = nil
		} else {
			err = fmt.Errorf("Passive client cannot be restarted")
			retry = false
			return
		}
	}

	client.started = true

	if client.conn == nil {
		client.log().Debug("Trying to establish a connection")

		if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
			err = connErr
			retry = true
			return
		} else {
			client.conn = conn
			client.address = conn.RemoteAddr().String()
		}
	}

	client.log().Info("Starting client")

	client.contactSent = false
	client.contactRecv = false
	client.initSent = false
	client.initRecv = false
	client.keepaliveStarted = false
	client.transferOutId = 0
	client.transferIn = nil

	client.msgsOut = make(chan Message, 100)
	client.msgsIn = make(chan Message, 100)
	client.transferOutSend = make(chan Message)
	client.transferOutAck = make(chan Message)

	client.handleMetaStop = make(chan struct{}, 10)
	client.handleMetaStopAck = make(chan struct{}, 2)
	client.handlerConnInStop = make(chan struct{}, 2)
	client.handlerConnInStopAck = make(chan struct{}, 2)
	client.handlerConnOutStop = make(chan struct{}, 2)
	client.handlerConnOutStopAck = make(chan struct{}, 2)
	client.handlerStateStop = make(chan struct{}, 2)
	client.handlerStateStopAck = make(chan struct{}, 2)

	client.reportChan = make(chan cla.ConvergenceStatus, 100)

	go client.handleMeta()
	go client.handleConnIn()
	go client.handleConnOut()
	go client.handleState()

	return
}

func (client *Client) Close() {
	client.handleMetaStop <- struct{}{}

	// TODO: there are currently some synchronization issues..
	select {
	case <-time.After(500 * time.Millisecond):
	case <-client.handleMetaStopAck:
	}
}

func (client *Client) Channel() chan cla.ConvergenceStatus {
	return client.reportChan
}

func (client *Client) Address() string {
	return client.address
}

func (client *Client) IsPermanent() bool {
	return client.permanent
}

func (client *Client) GetEndpointID() bundle.EndpointID {
	return client.endpointID
}

func (client *Client) GetPeerEndpointID() bundle.EndpointID {
	return client.peerEndpointID
}
