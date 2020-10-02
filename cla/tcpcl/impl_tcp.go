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

// TCPListener is a TCPCL server bound to a TCP port to accept incoming TCPCL connections.
// This type implements the cla.ConvergenceProvider and should be supervised by a cla.Manager.
type TCPListener struct {
	listenAddress string
	endpointID    bundle.EndpointID
	manager       *cla.Manager

	stopSyn chan struct{}
	stopAck chan struct{}
}

// ListenTCP creates a new TCPListener which should be bound to the given address and advertises the endpoint ID as
// its own node identifier.
func ListenTCP(listenAddress string, endpointID bundle.EndpointID) *TCPListener {
	return &TCPListener{
		listenAddress: listenAddress,
		endpointID:    endpointID,

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}
}

func (listener *TCPListener) RegisterManager(manager *cla.Manager) {
	listener.manager = manager
}

func (listener *TCPListener) Start() error {
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
				_ = ln.Close()
				close(listener.stopAck)

				return

			default:
				if err := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					log.WithError(err).WithField("cla", listener).Warn(
						"TCPListener failed to set deadline on TCP socket")

					listener.Close()
				} else if conn, err := ln.Accept(); err == nil {
					client := newClientTCP(conn, listener.endpointID)
					listener.manager.Register(client)
				}
			}
		}
	}(ln)

	return nil
}

func (listener *TCPListener) Close() {
	close(listener.stopSyn)
	<-listener.stopAck
}

func (listener TCPListener) String() string {
	return fmt.Sprintf("tcpcl://%s", listener.listenAddress)
}

func tcpClientStart(client *Client) error {
	if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
		return connErr
	} else {
		client.connReader = conn
		client.connWriter = conn
		client.connCloser = conn

		client.log().Debug("Dialed successfully")
		return nil
	}
}

// newClientTCP creates a new Client on an existing connection. This function is used from the TCPListener.
func newClientTCP(conn net.Conn, endpointID bundle.EndpointID) *Client {
	return &Client{
		address:         conn.RemoteAddr().String(),
		activePeer:      false,
		customStartFunc: tcpClientStart,
		connReader:      conn,
		connWriter:      conn,
		connCloser:      conn,
		nodeId:          endpointID,
	}
}

// DialTCP tries to establish a new TCPCL Client to a remote server.
func DialTCP(address string, endpointID bundle.EndpointID, permanent bool) *Client {
	return &Client{
		address:         address,
		permanent:       permanent,
		activePeer:      true,
		customStartFunc: tcpClientStart,
		nodeId:          endpointID,
	}
}
