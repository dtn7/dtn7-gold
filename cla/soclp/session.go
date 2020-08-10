// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type Session struct {
	in  io.Reader
	out io.Writer

	starFunc func() (error, bool)

	permanent    bool
	endpoint     bundle.EndpointID
	peerEndpoint bundle.EndpointID

	statusChannel chan cla.ConvergenceStatus
}

func (s *Session) Close() {
	panic("implement me")
}

// Start this Session. In case of an error, retry indicates that another try should be made later.
func (s *Session) Start() (err error, retry bool) {
	s.peerEndpoint = bundle.EndpointID{}
	s.statusChannel = make(chan cla.ConvergenceStatus)

	return s.starFunc()
}

func (s *Session) Channel() chan cla.ConvergenceStatus {
	return s.statusChannel
}

func (s *Session) Address() string {
	panic("implement me")
}

// IsPermanent returns true, if this CLA should not be removed after failures.
func (s *Session) IsPermanent() bool {
	return s.permanent
}

func (s *Session) Send(bndl *bundle.Bundle) error {
	panic("implement me")
}

// GetEndpointID returns this instance's endpoint identifier.
func (s *Session) GetEndpointID() bundle.EndpointID {
	return s.endpoint
}

// GetPeerEndpointID returns the peer's endpoint identifier, if known. Otherwise, dtn:none will be returned.
func (s *Session) GetPeerEndpointID() bundle.EndpointID {
	if s.peerEndpoint == (bundle.EndpointID{}) {
		return bundle.DtnNone()
	} else {
		return s.peerEndpoint
	}
}

// logger returns a new logrus.Entry.
func (s *Session) logger() (e *log.Entry) {
	e = log.WithField("soclp-session", s.Address())
	if s.peerEndpoint != (bundle.EndpointID{}) {
		e = e.WithField("peer", s.peerEndpoint)
	}

	return
}
