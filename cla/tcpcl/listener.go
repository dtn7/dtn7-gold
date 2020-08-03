// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// Listener is a TCPCL server bound to a TCP port to accept incoming TCPCL connections.
// This type implements the cla.ConvergenceProvider and should be supervised by a cla.Manager.
type Listener struct {
	listenAddress string
	endpointID    bundle.EndpointID
	manager       *cla.Manager
	clas          []cla.Convergence

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewListener creates a new Listener which should be bound to the given address and advertises the endpoint ID as
// its own node identifier.
func NewListener(listenAddress string, endpointID bundle.EndpointID) *Listener {
	return &Listener{
		listenAddress: listenAddress,
		endpointID:    endpointID,

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}
}

func (listener *Listener) RegisterManager(manager *cla.Manager) {
	listener.manager = manager
}

func (listener *Listener) Start() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", listener.listenAddress)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	go func(ln *net.TCPListener) {
		for {
			select {
			case <-listener.stopSyn:
				ln.Close()
				close(listener.stopAck)

				return

			default:
				if err := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					log.WithError(err).WithField("cla", listener).Warn(
						"Listener failed to set deadline on TCP socket")

					listener.Close()
				} else if conn, err := ln.Accept(); err == nil {
					client := NewClient(conn, listener.endpointID)
					listener.clas = append(listener.clas, client)
					listener.manager.Register(client)
				}
			}
		}
	}(ln)

	return nil
}

func (listener *Listener) Close() {
	close(listener.stopSyn)
	<-listener.stopAck
}

func (listener Listener) String() string {
	return fmt.Sprintf("tcpcl://%s", listener.listenAddress)
}
