// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func (s *Session) handleIn() {
	defer func() {
		// TODO
	}()

	for {
		var message Message
		if err := cboring.Unmarshal(&message, s.in); err != nil {
			s.logger().WithError(err).Error("Unmarshalling CBOR message errored")
			return
		}

		var msgErr error
		switch msg := message.MessageType.(type) {
		case *IdentityMessage:
			msgErr = s.receiveIdentity(msg)

		case *StatusMessage:
			msgErr = s.receiveStatus(msg)

		case *TransferMessage:
			msgErr = s.receiveTransfer(msg)

		case *TransferAckMessage:

		default:
			msgErr = fmt.Errorf("unsupported message type %T", msg)
		}

		if msgErr != nil {
			s.logger().WithError(msgErr).WithField("message", message).Error("Handling received message errored")
			return
		}
	}
}

// receiveIdentity sets the peer's endpoint ID if not configured yet.
func (s *Session) receiveIdentity(im *IdentityMessage) (err error) {
	if s.peerEndpoint != (bundle.EndpointID{}) {
		return fmt.Errorf("peer endpoint ID is already configured")
	}

	s.peerEndpoint = im.NodeID
	s.Channel() <- cla.NewConvergencePeerAppeared(s, im.NodeID)

	s.logger().WithField("peer", im.NodeID).Info("Established handshake with peer")

	return
}

// receiveStatus inspects incoming status messages.
func (s *Session) receiveStatus(sm *StatusMessage) (err error) {
	switch status := sm.StatusCode; status {
	case StatusShutdown:
		// TODO
	case StatusHeartbeat:
		// TODO
	default:
		err = fmt.Errorf("unsupported status message code %d", status)
	}

	return
}

func (s *Session) receiveTransfer(tm *TransferMessage) (err error) {
	s.Channel() <- cla.NewConvergenceReceivedBundle(s, s.GetEndpointID(), &tm.Bundle)
	// TODO: send ack

	s.logger().WithField("bundle", tm.Bundle.String()).Info("Received bundle")

	return
}

func (s *Session) receiveTransferAck(am *TransferAckMessage) (err error) {
	// TODO
	return
}
