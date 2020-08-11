// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func (s *Session) handleIn() {
	defer s.closeAction()

	for {
		var message Message
		if err := cboring.Unmarshal(&message, s.In); err != nil {
			if err == io.EOF || err == io.ErrClosedPipe {
				s.logger().WithError(err).Debug("Input stream reached its end")
			} else {
				s.logger().WithError(err).Error("Unmarshalling CBOR message errored")
			}

			return
		}

		s.logger().WithField("message", message).Debug("Received incoming message")

		s.updateLastReceive()

		var msgErr error
		switch msg := message.MessageType.(type) {
		case *IdentityMessage:
			msgErr = s.receiveIdentity(msg)

		case *StatusMessage:
			msgErr = s.receiveStatus(msg)

		case *TransferMessage:
			msgErr = s.receiveTransfer(msg)

		case *TransferAckMessage:
			msgErr = s.receiveTransferAck(msg)

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
	s.peerEndpointLock.Lock()

	if s.peerEndpoint != (bundle.EndpointID{}) {
		return fmt.Errorf("peer endpoint ID is already configured")
	}

	s.peerEndpoint = im.NodeID
	s.peerEndpointLock.Unlock()

	s.Channel() <- cla.NewConvergencePeerAppeared(s, im.NodeID)

	s.logger().WithField("peer", im.NodeID).Info("Established handshake with peer")

	return
}

// receiveStatus inspects incoming status messages.
func (s *Session) receiveStatus(sm *StatusMessage) (err error) {
	switch status := sm.StatusCode; status {
	case StatusShutdown:
		s.logger().Info("Received shutdown status message")
		go s.closeAction()

	case StatusHeartbeat:
		// TODO

	default:
		err = fmt.Errorf("unsupported status message code %d", status)
	}

	return
}

// receiveTransfer handles with incoming transfers.
func (s *Session) receiveTransfer(tm *TransferMessage) (err error) {
	s.Channel() <- cla.NewConvergenceReceivedBundle(s, s.GetEndpointID(), &tm.Bundle)

	s.outChannel <- Message{MessageType: NewTransferAckMessage(tm.Identifier)}

	s.logger().WithFields(log.Fields{
		"bundle":      tm.Bundle.String(),
		"transfer-id": tm.Identifier,
	}).Info("Received bundle")

	return
}

// receiveTransferAck inspects incoming transfer acknowledgements.
func (s *Session) receiveTransferAck(am *TransferAckMessage) (err error) {
	s.transferAcks.Store(am.Identifier, struct{}{})

	s.logger().WithField("transfer-id", am.Identifier).Info("Received reception acknowledge")
	return
}

// updateLastReceive sets lastReceive to the current time.
func (s *Session) updateLastReceive() {
	s.lastReceiveLock.Lock()
	defer s.lastReceiveLock.Unlock()

	s.lastReceive = time.Now()
	s.logger().WithField("last-receive", s.lastReceive).Debug("Updated last receive timestamp")
}
