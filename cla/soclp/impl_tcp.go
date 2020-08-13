// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// newTcpConnSession based on a net.Conn and a local Endpoint ID.
//
// One might want to alter the Restartable and Permanent field. Further, the StartFunc should be modified for "dial in".
func newTcpConnSession(conn net.Conn, endpointID bundle.EndpointID) *Session {
	addrFunc := func(s *Session) string {
		if s == nil {
			return "none"
		} else if sConn, ok := s.In.(net.Conn); !ok {
			return "invalid TCP session"
		} else {
			return fmt.Sprintf("soclp-tcp:%v", sConn.RemoteAddr())
		}
	}

	return &Session{
		In:               conn,
		Out:              conn,
		Closer:           conn,
		StartFunc:        nil,
		AddressFunc:      addrFunc,
		Permanent:        false,
		Endpoint:         endpointID,
		HeartbeatTimeout: 30 * time.Second,
	}
}

// DialTcp creates a new TCP-based Session.
func DialTcp(address string, endpointID bundle.EndpointID, permanent bool) (s *Session) {
	s = newTcpConnSession(nil, endpointID)
	s.Restartable = true
	s.StartFunc = func(s *Session) (err error, retry bool) {
		if conn, connErr := net.DialTimeout("tcp", address, time.Second); connErr != nil {
			return connErr, true
		} else {
			s.In = conn
			s.Out = conn
			s.Closer = conn

			return
		}
	}
	s.Permanent = permanent

	return
}

// TcpListener is a cla.ConvergenceProvider which listens on a TCP port and handles multiple TCP-based SoCLP sessions.
type TcpListener struct {
	listenAddress string
	endpointID    bundle.EndpointID
	manager       *cla.Manager

	closeSyn chan struct{}
	closeAck chan struct{}
}

// NewTcpListener for an address to listen on (e.g. ":2323" or "localhost:2323") and a self-identifying endpoint ID.
func NewTcpListener(listenAddress string, endpointID bundle.EndpointID) *TcpListener {
	return &TcpListener{
		listenAddress: listenAddress,
		endpointID:    endpointID,
	}
}

// RegisterManager for convergence reporting.
func (l *TcpListener) RegisterManager(manager *cla.Manager) {
	l.manager = manager
}

// Start this TcpListener to deal with incoming connections.
func (l *TcpListener) Start() error {
	l.closeSyn = make(chan struct{})
	l.closeAck = make(chan struct{})

	if l.manager == nil {
		return fmt.Errorf("no manager is configured")
	}

	if tcpAddr, tcpAddrErr := net.ResolveTCPAddr("tcp", l.listenAddress); tcpAddrErr != nil {
		return tcpAddrErr
	} else if ln, lnErr := net.ListenTCP("tcp", tcpAddr); lnErr != nil {
		return lnErr
	} else {
		go l.handler(ln)
	}

	return nil
}

// handler for the TcpListener's tasks.
func (l *TcpListener) handler(ln *net.TCPListener) {
	logger := log.WithField("cla", l)
	logger.Info("Starting TCP-based SoCLP listener")

	defer func() {
		logger.Info("Closing down TCP-based SoCLP listener")
		close(l.closeAck)

		if err := ln.Close(); err != nil {
			logger.WithError(err).Warn("Closing TCP listener errored")
		}
	}()

	for {
		select {
		case <-l.closeSyn:
			return

		default:
			if deadlineErr := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); deadlineErr != nil {
				logger.WithError(deadlineErr).Error("Setting deadline on TCP socket errored")
				return
			} else if conn, connErr := ln.Accept(); connErr == nil {
				session := newTcpConnSession(conn, l.endpointID)
				session.Restartable = false
				l.manager.Register(session)
			}
		}
	}
}

// Close down this TcpListener and all its connections.
func (l *TcpListener) Close() {
	close(l.closeSyn)
	<-l.closeAck
}

func (l *TcpListener) String() string {
	return "soclp-tcp://TODO"
}
