// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpclv4

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/utils"
)

// TCPListener is a TCPCLv4 server bound to a TCP port to accept incoming TCPCLv4 connections.
//
// This type implements the cla.ConvergenceProvider and should be supervised by a cla.Manager.
type TCPListener struct {
	listenAddress string
	endpointID    bpv7.EndpointID
	manager       *cla.Manager

	stopSyn chan struct{}
	stopAck chan struct{}
}

// ListenTCP creates a new TCPListener which should be bound to the given address and advertises the endpoint iD as
// its own node identifier.
func ListenTCP(listenAddress string, endpointID bpv7.EndpointID) *TCPListener {
	return &TCPListener{
		listenAddress: listenAddress,
		endpointID:    endpointID,

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}
}

// RegisterManager tells the TCPListener where to report new instances of cla.Convergence to.
func (listener *TCPListener) RegisterManager(manager *cla.Manager) {
	listener.manager = manager
}

// Start this TCPListener. Before being started, the the RegisterManager method tells this Client its cla.Manager. The
// cla.Manager will both call the RegisterManager and Start methods.
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
					log.WithError(err).WithField("cla", listener).Error(
						"TCPListener failed to set deadline on TCP socket")

					_ = listener.Close()
				} else if conn, err := ln.Accept(); err == nil {
					client := newClientTCP(conn, listener.endpointID)
					listener.manager.Register(client)
				}
			}
		}
	}(ln)

	return nil
}

// Close signals this TCPListener to shut down.
func (listener *TCPListener) Close() error {
	close(listener.stopSyn)
	<-listener.stopAck

	return nil
}

func (listener TCPListener) String() string {
	return fmt.Sprintf("tcpclv4://%s", listener.listenAddress)
}

// tcpClientStart is the Client's customStartFunc for TCP.
func tcpClientStart(client *Client) error {
	if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
		return connErr
	} else {
		client.connCloser = conn
		client.messageSwitch = utils.NewMessageSwitchReaderWriter(conn, conn)

		client.log().Debug("Dialed successfully")
		return nil
	}
}

// newClientTCP creates a new Client on an existing connection. This function is used from the TCPListener.
func newClientTCP(conn net.Conn, endpointID bpv7.EndpointID) *Client {
	return &Client{
		address:         conn.RemoteAddr().String(),
		activePeer:      false,
		customStartFunc: tcpClientStart,
		connCloser:      conn,
		messageSwitch:   utils.NewMessageSwitchReaderWriter(conn, conn),
		nodeId:          endpointID,
	}
}

// DialTCP tries to establish a new TCPCLv4 Client to a remote TCPListener.
func DialTCP(address string, endpointID bpv7.EndpointID, permanent bool) *Client {
	return &Client{
		address:         address,
		permanent:       permanent,
		activePeer:      true,
		customStartFunc: tcpClientStart,
		nodeId:          endpointID,
	}
}
