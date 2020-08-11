// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type Session struct {
	// In and Out are the streams to operate on.
	In  io.Reader
	Out io.Writer

	// Closer closes the underlying stream, might be nil.
	Closer io.Closer

	// StartFunc represents additional startup code, e.g., to establish a TCP connection. Might be nil.
	StartFunc func() (error, bool)

	// AddressFunc generates this Session's Address.
	AddressFunc func() string

	// Permanent is true iff this Session should be permanent resp. not be removed on connection issues.
	Permanent bool

	// Endpoint is this node's Endpoint ID; this node, not the peer.
	Endpoint bundle.EndpointID

	// peerEndpoint is the Endpoint ID of the peer and has its mutex for read/write access.
	peerEndpoint     bundle.EndpointID
	peerEndpointLock sync.Mutex

	// SendTimeout is the maximum time for sending an outgoing Bundle.
	SendTimeout time.Duration

	// statusChannel is outgoing, see Channel().
	statusChannel chan cla.ConvergenceStatus

	// outChannel and outStopChannel are to communicate with the outgoing handler
	outChannel     chan Message
	outStopChannel chan struct{}

	// transferAcks stores received ack identifiers, sync.Map[uint64]struct{}
	transferAcks sync.Map

	closeOnce sync.Once
}

// closeAction performs the closing within a sync.Once.
//
// This method is called from the exported Close method, which starts with sending a Shutdown StatusMessage.
func (s *Session) closeAction() {
	s.closeOnce.Do(func() {
		s.logger().Info("Closing down")

		close(s.outStopChannel)

		if s.Closer != nil {
			if err := s.Closer.Close(); err != nil {
				s.logger().WithError(err).Warn("Closing down errored")
			}
		}
	})
}

// Close down this session and try telling the peer to do the same.
func (s *Session) Close() {
	if s.outChannel != nil {
		s.outChannel <- Message{NewShutdownStatusMessage()}
	}

	s.closeAction()
}

// Start this Session. In case of an error, retry indicates that another try should be made later.
func (s *Session) Start() (err error, retry bool) {
	s.peerEndpoint = bundle.EndpointID{}
	s.peerEndpointLock = sync.Mutex{}
	s.statusChannel = make(chan cla.ConvergenceStatus)
	s.outChannel = make(chan Message)
	s.outStopChannel = make(chan struct{})
	s.transferAcks = sync.Map{}
	s.closeOnce = sync.Once{}

	if s.StartFunc != nil {
		if err, retry = s.StartFunc(); err != nil {
			return
		}
	}

	go s.handleIn()
	go s.handleOut()

	s.outChannel <- Message{NewIdentityMessage(s.Endpoint)}

	return
}

// Channel for status information and received Bundles.
func (s *Session) Channel() chan cla.ConvergenceStatus {
	return s.statusChannel
}

// Address for this Session's instance, should be kind of unique.
func (s *Session) Address() string {
	return s.AddressFunc()
}

// IsPermanent returns true, if this CLA should not be removed after failures.
func (s *Session) IsPermanent() bool {
	return s.Permanent
}

// Send a Bundle to the peer.
func (s *Session) Send(b *bundle.Bundle) error {
	if tm, tmErr := NewTransferMessage(*b); tmErr != nil {
		return tmErr
	} else {
		s.outChannel <- Message{MessageType: tm}

		for timeout := time.Now().Add(s.SendTimeout); time.Now().Before(timeout); {
			if _, ack := s.transferAcks.Load(tm.Identifier); ack {
				s.transferAcks.Delete(tm.Identifier)
				return nil
			}

			time.Sleep(50 * time.Millisecond)
		}

		return fmt.Errorf("waiting for acknowledgement timed out after %v", s.SendTimeout)
	}
}

// GetEndpointID returns this instance's endpoint identifier.
func (s *Session) GetEndpointID() bundle.EndpointID {
	return s.Endpoint
}

// GetPeerEndpointID returns the peer's endpoint identifier, if known. Otherwise, dtn:none will be returned.
func (s *Session) GetPeerEndpointID() bundle.EndpointID {
	s.peerEndpointLock.Lock()
	defer s.peerEndpointLock.Unlock()

	if s.peerEndpoint == (bundle.EndpointID{}) {
		return bundle.DtnNone()
	} else {
		return s.peerEndpoint
	}
}

// logger returns a new logrus.Entry.
func (s *Session) logger() (e *log.Entry) {
	e = log.WithField("soclp-session", s.Address())

	if peer := s.GetPeerEndpointID(); peer != bundle.DtnNone() {
		e = e.WithField("peer", peer)
	}

	return
}
