// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpclv4

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/tcpclv4/internal/stages"
	"github.com/dtn7/dtn7-go/cla/tcpclv4/internal/utils"
)

// Client is a TCPCL client for a bidirectional Bundle exchange. Thus, the Client type implements both
// cla.ConvergenceReceiver and cla.ConvergenceSender. A Client can be created by the TCPListener for incoming
// connections or dialed for outgoing connections.
type Client struct {
	address    string
	permanent  bool
	activePeer bool

	customStartFunc func(*Client) error

	started    bool
	connReader io.Reader
	connWriter io.Writer
	connCloser io.Closer

	messageSwitch   *utils.MessageSwitch
	stageHandler    *stages.StageHandler
	transferManager *utils.TransferManager

	nodeId     bundle.EndpointID
	peerNodeId bundle.EndpointID

	reportChan chan cla.ConvergenceStatus

	closeChanSyn chan struct{}
	closeChanAck chan struct{}
}

func (client *Client) String() string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "TCPCL(")
	_, _ = fmt.Fprintf(&b, "address=%s, ", client.Address())
	_, _ = fmt.Fprintf(&b, "node=%s, ", client.GetEndpointID())
	_, _ = fmt.Fprintf(&b, "peer=%v, ", client.GetPeerEndpointID())
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
			client.connReader = nil
			client.connWriter = nil
			client.connCloser = nil
		} else {
			err = fmt.Errorf("passive client cannot be restarted")
			retry = false
			return
		}
	}

	client.started = true

	if client.connReader == nil {
		if err = client.customStartFunc(client); err != nil {
			retry = true
			return
		}
	}

	client.reportChan = make(chan cla.ConvergenceStatus, 32)

	client.closeChanSyn = make(chan struct{})
	client.closeChanAck = make(chan struct{})

	client.messageSwitch = utils.NewMessageSwitch(client.connReader, client.connWriter)
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
			PreHook: func(_ *stages.StageHandler, _ *stages.State) error {
				client.log().Debug("Starting Contact Stage")
				return nil
			},
		},
		{
			Stage: &stages.SessInitStage{},
			PreHook: func(_ *stages.StageHandler, _ *stages.State) error {
				client.log().Debug("Starting Session Init Stage")
				return nil
			},
			PostHook: func(_ *stages.StageHandler, state *stages.State) error {
				client.peerNodeId = state.PeerNodeId
				return nil
			},
		},
		{
			Stage: &stages.SessEstablishedStage{},
			PreHook: func(_ *stages.StageHandler, state *stages.State) error {
				client.log().Debug("Starting Session Established Stage")

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
		stageHandlerIn, stageHandlerOut := client.stageHandler.Exchanges()
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

		closeErrFuncs := []func() error{
			client.transferManager.Close,
			client.stageHandler.Close,
			client.messageSwitch.Close,
			func() error {
				if client.connCloser != nil {
					return client.connCloser.Close()
				} else {
					return nil
				}
			},
		}
		for i, errFunc := range closeErrFuncs {
			if err := errFunc(); err != nil {
				client.log().WithError(err).WithField("no", i).Debug("Error occurred while closing")
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
			if errors.Is(err, io.EOF) {
				client.log().Info("Received EOF")
			} else {
				client.log().WithError(err).Error("Error occurred")
			}
			return
		}
	}
}

func (client *Client) Send(b *bundle.Bundle) error {
	client.log().WithField("bundle", *b).Debug("Sending Bundle...")
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
