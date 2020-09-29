// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/stages"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/utils"
)

// Client is a TCPCL client for a bidirectional Bundle exchange. Thus, the Client type implements both
// cla.ConvergenceReceiver and cla.ConvergenceSender. A Client can be created by the Listener for incoming
// connections or dialed for outgoing connections.
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

	closeChanSyn chan struct{}
	closeChanAck chan struct{}
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
	if client.activePeer {
		_, _ = fmt.Fprintf(&b, "peer=active")
	} else {
		_, _ = fmt.Fprintf(&b, "peer=passive")
	}
	_, _ = fmt.Fprintf(&b, ")")

	return b.String()
}

func (client *Client) log() *log.Entry {
	return log.WithField("cla", client.String())
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

			client.log().Debug("Dialed successfully")
		}
	}

	client.reportChan = make(chan cla.ConvergenceStatus, 32)

	client.closeChanSyn = make(chan struct{})
	client.closeChanAck = make(chan struct{})

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
		{
			Stage: &stages.ContactStage{},
			StartHook: func(_ *stages.StageHandler, _ *stages.State) error {
				client.log().Debug("Started Contact Stage")
				return nil
			},
		},
		{
			Stage: &stages.SessInitStage{},
			StartHook: func(_ *stages.StageHandler, _ *stages.State) error {
				client.log().Debug("Started Session Init Stage")
				return nil
			},
			PostHook: func(_ *stages.StageHandler, state *stages.State) error {
				client.peerNodeId = state.PeerNodeId
				return nil
			},
		},
		{
			Stage: &stages.SessEstablishedStage{},
			StartHook: func(_ *stages.StageHandler, state *stages.State) error {
				client.log().Debug("Started Session Established Stage")

				sMtuChan <- state.SegmentMtu
				return nil
			},
		}}
	client.stageHandler = stages.NewStageHandler(stageHandlerStages, msIncoming, msOutgoing, conf)

	select {
	case <-time.After(15 * time.Second):
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

	client.log().Info("Started TCPCL4")

	client.reportChan <- cla.NewConvergencePeerAppeared(client, client.peerNodeId)

	go client.handle()
	return
}

func (client *Client) handle() {
	_, _, messageSwitchErr := client.messageSwitch.Exchange()
	stageHandlerErr := client.stageHandler.Error()
	incomingBundles, transferManagerErr := client.transferManager.Exchange()

	defer func() {
		client.log().Info("Closing down TCPCL4")

		client.reportChan <- cla.NewConvergencePeerDisappeared(client, client.peerNodeId)

		closeErrs := []error{
			client.transferManager.Close(),
			client.stageHandler.Close(),
			client.messageSwitch.Close()}
		for i, err := range closeErrs {
			if err != nil {
				client.log().WithError(err).WithField("no", i).Warn("Error occurred while closing")
			}
		}

		client.transferManager = nil
		client.stageHandler = nil
		client.messageSwitch = nil

		close(client.closeChanAck)
	}()

	for {
		var err error
		select {
		case b := <-incomingBundles:
			client.log().WithField("bundle", b).Info("Received Bundle")
			client.reportChan <- cla.NewConvergenceReceivedBundle(client, client.nodeId, &b)

		case <-client.closeChanSyn:
			client.log().Debug("Received close signal")
			return

		case err = <-messageSwitchErr:
		case err = <-stageHandlerErr:
		case err = <-transferManagerErr:
		}

		if err != nil {
			client.log().WithError(err).Error("Error occurred")
			return
		}
	}
}

func (client *Client) Send(b *bundle.Bundle) error {
	client.log().WithField("bundle", *b).Info("Sending Bundle...")
	defer client.log().WithField("bundle", *b).Info("Sent Bundle")

	return client.transferManager.Send(*b)
}

func (client *Client) Close() {
	close(client.closeChanSyn)
	<-client.closeChanAck
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
