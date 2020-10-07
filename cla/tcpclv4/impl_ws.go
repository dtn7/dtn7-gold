// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpclv4

import (
	"net/http"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/tcpclv4/internal/utils"
)

// WebSocketListener is a TCPCLv4 server as a http.Handler to accept incoming TCPCLv4 connections via WebSockets.
//
// This type implements the cla.ConvergenceProvider and should be supervised by a cla.Manager.
type WebSocketListener struct {
	endpointID bundle.EndpointID

	manager      *cla.Manager
	managerReady uint32

	upgrader websocket.Upgrader
}

// ListenWebSocket creates a new WebSocketListener.
func ListenWebSocket(endpointID bundle.EndpointID) *WebSocketListener {
	return &WebSocketListener{
		endpointID: endpointID,
		upgrader:   websocket.Upgrader{},
	}
}

// RegisterManager tells the WebSocketListener where to report new instances of cla.Convergence to.
func (listener *WebSocketListener) RegisterManager(manager *cla.Manager) {
	listener.manager = manager
	atomic.StoreUint32(&listener.managerReady, 1)
}

// Start this WebSocketListener.
func (listener *WebSocketListener) Start() error {
	// There is no work to be done here. The heavy lifting is outsourced to the underlying http.Server.
	return nil
}

// Close this WebSocketListener.
func (listener *WebSocketListener) Close() error {
	// Again, there is nothing to do here.
	return nil
}

// ServeHTTP upgrades a HTTP connection to a WebSocket connection which is used for TCPCLv4.
func (listener *WebSocketListener) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if atomic.LoadUint32(&listener.managerReady) != 1 {
		return
	}

	if conn, err := listener.upgrader.Upgrade(writer, request, nil); err != nil {
		log.WithField("cla", listener).WithError(err).Warn("Upgrading connection errored")
	} else {
		client := newClientWebSocket(conn, listener.endpointID)
		listener.manager.Register(client)
	}
}

// webSocketClientStart is the Client's customStartFunc for WebSockets.
func webSocketClientStart(client *Client) error {
	if conn, _, err := websocket.DefaultDialer.Dial(client.address, nil); err != nil {
		return err
	} else {
		rwc := utils.NewWebSocketReadWriteFlushCloser(conn)
		client.connReader = rwc
		client.connWriter = rwc
		client.connCloser = rwc

		client.log().Debug("Dialed successfully")
		return nil
	}
}

// newClientWebSocket creates a new Client on a new *websocket.Conn. This function is called from the WebSocketListener.
func newClientWebSocket(conn *websocket.Conn, endpointID bundle.EndpointID) *Client {
	rwc := utils.NewWebSocketReadWriteFlushCloser(conn)
	return &Client{
		address:         conn.RemoteAddr().String(),
		activePeer:      false,
		customStartFunc: webSocketClientStart,
		connReader:      rwc,
		connWriter:      rwc,
		connCloser:      rwc,
		nodeId:          endpointID,
	}
}

// DialWebSocket tries to establish a new TCPCLv4 Client to a remote WebSocketListener.
func DialWebSocket(address string, endpointID bundle.EndpointID, permanent bool) *Client {
	return &Client{
		address:         address,
		permanent:       permanent,
		activePeer:      true,
		customStartFunc: webSocketClientStart,
		nodeId:          endpointID,
	}
}
