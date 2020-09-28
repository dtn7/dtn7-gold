// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/stages"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/utils"
)

// Client is a TCPCL client for a bidirectional Bundle exchange. Thus, the Client type implements both
// cla.ConvergenceReceiver and cla.ConvergenceSender. A Client can be created by the Listener for incoming
// connections or dialed for outbounding connections.
type Client struct {
	address    string
	permanent  bool
	activePeer bool

	started bool
	conn    net.Conn

	messageSwitch   *utils.MessageSwitch
	stageHandler    *stages.StageHandler
	transferManager *utils.TransferManager

	nodeId     bundle.EndpointID
	peerNodeId bundle.EndpointID

	reportChan chan cla.ConvergenceStatus
	closeChan  chan struct{}
}

// NewClient creates a new Client on an existing connection. This function is used from the Listener.
func NewClient(conn net.Conn, endpointID bundle.EndpointID) *Client {
	return &Client{
		address:    conn.RemoteAddr().String(),
		activePeer: false,
		conn:       conn,
		nodeId:     endpointID,
	}
}

// DialClient tries to establish a new TCPCL Client to a remote server.
func DialClient(address string, endpointID bundle.EndpointID, permanent bool) *Client {
	return &Client{
		address:    address,
		permanent:  permanent,
		activePeer: true,
		nodeId:     endpointID,
	}
}

func (client *Client) String() string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "TCPCL(")
	if client.conn != nil {
		_, _ = fmt.Fprintf(&b, "peer=%v, ", client.conn.RemoteAddr())
	} else {
		_, _ = fmt.Fprintf(&b, "peer=NONE, ")
	}
	_, _ = fmt.Fprintf(&b, "activePeer peer=%t", client.activePeer)
	_, _ = fmt.Fprintf(&b, ")")

	return b.String()
}

func (client *Client) Start() (err error, retry bool) {
	if client.started {
		if client.activePeer {
			client.conn = nil
		} else {
			err = fmt.Errorf("passive client cannot be restarted")
			retry = false
			return
		}
	}

	client.started = true

	if client.conn == nil {
		if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
			err = connErr
			retry = true
			return
		} else {
			client.conn = conn
			client.address = conn.RemoteAddr().String()
		}
	}

	client.reportChan = make(chan cla.ConvergenceStatus, 32)
	client.closeChan = make(chan struct{})

	client.messageSwitch = utils.NewMessageSwitch(client.conn, client.conn)
	msIncoming, msOutgoing, _ := client.messageSwitch.Exchange()

	conf := stages.Configuration{
		ActivePeer:   client.activePeer,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   52428800,
		TransferMru:  1073741824,
		NodeId:       client.nodeId,
	}

	sMtuChan := make(chan uint64)
	stageHandlerStages := []stages.StageSetup{
		{Stage: &stages.ContactStage{}},
		{
			Stage: &stages.SessInitStage{},
			PostHook: func(_ *stages.StageHandler, state *stages.State) error {
				client.peerNodeId = state.PeerNodeId
				return nil
			},
		},
		{
			Stage: &stages.SessEstablishedStage{},
			StartHook: func(_ *stages.StageHandler, state *stages.State) error {
				sMtuChan <- state.SegmentMtu
				return nil
			},
		}}
	client.stageHandler = stages.NewStageHandler(stageHandlerStages, msIncoming, msOutgoing, conf)

	select {
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("establishing an exchangable connection timed out")
		retry = true
		return

	case sMtu := <-sMtuChan:
		var stageHandlerOut chan<- msgs.Message
		var stageHandlerIn <-chan msgs.Message
		var stageHandlerOk bool

		for i := 0; i < 5; i++ {
			if stageHandlerOut, stageHandlerIn, stageHandlerOk = client.stageHandler.Exchanges(); stageHandlerOk {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !stageHandlerOk {
			err = fmt.Errorf("fetching exchange channels failed")
			retry = true
			return
		}

		client.transferManager = utils.NewTransferManager(stageHandlerIn, stageHandlerOut, sMtu)
	}

	client.reportChan <- cla.NewConvergencePeerAppeared(client, client.peerNodeId)

	go client.handle()
	return
}

func (client *Client) handle() {
	_, _, messageSwitchErr := client.messageSwitch.Exchange()
	stageHandlerErr := client.stageHandler.Error()
	incomingBundles, transferManagerErr := client.transferManager.Exchange()

	defer func() {
		client.reportChan <- cla.NewConvergencePeerDisappeared(client, client.peerNodeId)

		_ = client.transferManager.Close()
		_ = client.stageHandler.Close()
		_ = client.messageSwitch.Close()

		client.transferManager = nil
		client.stageHandler = nil
		client.messageSwitch = nil
	}()

	for {
		var err error
		select {
		case b := <-incomingBundles:
			client.reportChan <- cla.NewConvergenceReceivedBundle(client, client.nodeId, &b)

		case <-client.closeChan:
			return

		case err = <-messageSwitchErr:
		case err = <-stageHandlerErr:
		case err = <-transferManagerErr:
		}

		if err != nil {
			// TODO
			return
		}
	}
}

func (client *Client) Send(b *bundle.Bundle) error {
	return client.transferManager.Send(*b)
}

func (client *Client) Close() {
	close(client.closeChan)
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
	return client.nodeId
}

func (client *Client) GetPeerEndpointID() bundle.EndpointID {
	return client.peerNodeId
}
