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

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/stages"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/utils"
)

// Client is a TCPCLv4 client for a bidirectional Bundle exchange. Thus, the Client type implements both
// cla.ConvergenceReceiver and cla.ConvergenceSender.
//
// A Client can be created by a Listener, e.g., a TCPListener, for incoming connections or dialed for outgoing
// connections, e.g., via DialTCP.
type Client struct {
	address    string
	permanent  bool
	activePeer bool

	customStartFunc func(*Client) error

	started    bool
	connCloser io.Closer

	messageSwitch   utils.MessageSwitch
	stageHandler    *stages.StageHandler
	transferManager *utils.TransferManager

	nodeId     bpv7.EndpointID
	peerNodeId bpv7.EndpointID

	reportChan chan cla.ConvergenceStatus

	closeChanSyn chan struct{}
	closeChanAck chan struct{}
}

func (client *Client) String() string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "TCPCLv4(")
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

// Start this Client and return both an error and a boolean indicating if another Start should be tried later.
func (client *Client) Start() (err error, retry bool) {
	if client.started {
		if client.activePeer {
			client.connCloser = nil
		} else {
			err = fmt.Errorf("passive client cannot be restarted")
			retry = false
			return
		}
	}

	client.started = true

	client.reportChan = make(chan cla.ConvergenceStatus, 32)

	client.closeChanSyn = make(chan struct{})
	client.closeChanAck = make(chan struct{})

	if client.messageSwitch == nil {
		if err = client.customStartFunc(client); err != nil {
			retry = true
			return
		}
	}
	msIncoming, msOutgoing, _ := client.messageSwitch.Exchange()

	conf := stages.Configuration{
		ActivePeer:   client.activePeer,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   1048576,
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

	client.log().Info("Started TCPCLv4")

	client.reportChan <- cla.NewConvergencePeerAppeared(client, client.peerNodeId)

	go client.handle()
	return
}

func (client *Client) handle() {
	_, _, messageSwitchErr := client.messageSwitch.Exchange()
	stageHandlerErr := client.stageHandler.Error()
	incomingBundles, transferManagerErr := client.transferManager.Exchange()

	defer func() {
		client.log().Info("Closing down TCPCLv4")

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

// Send a bundle to this Client's endpoint.
func (client *Client) Send(b bpv7.Bundle) error {
	client.log().WithField("bundle", b).Debug("Sending Bundle...")
	defer client.log().WithField("bundle", b).Info("Sent Bundle")

	return client.transferManager.Send(b)
}

// Close signals this Client to shut down.
func (client *Client) Close() error {
	close(client.closeChanSyn)
	<-client.closeChanAck

	return nil
}

// Channel represents a return channel for transmitted bundles, status messages, etc.
func (client *Client) Channel() chan cla.ConvergenceStatus {
	return client.reportChan
}

// Address should return a unique address string to both identify this Client and ensure it will not opened twice.
func (client *Client) Address() string {
	return client.address
}

// IsPermanent returns true, if this CLA should not be removed after failures.
func (client *Client) IsPermanent() bool {
	return client.permanent
}

// GetEndpointID returns the endpoint iD assigned to this CLA.
func (client *Client) GetEndpointID() bpv7.EndpointID {
	return client.nodeId
}

// GetPeerEndpointID returns the endpoint iD assigned to this CLA's peer, if it's known. Otherwise the zero endpoint
// will be returned.
func (client *Client) GetPeerEndpointID() bpv7.EndpointID {
	return client.peerNodeId
}
