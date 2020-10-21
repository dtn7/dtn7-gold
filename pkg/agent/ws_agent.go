// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// WebSocketAgent is a WebSocket based ApplicationAgent. It can be used together with the WebSocketAgentConnector to
// exchange Messages.
type WebSocketAgent struct {
	receiver  chan Message
	clientMux *MuxAgent

	upgrader websocket.Upgrader
}

// NewWebSocketAgent will be started with its handler. The ServeHTTP function must be bound to the HTTP server.
func NewWebSocketAgent() (wa *WebSocketAgent) {
	wa = &WebSocketAgent{
		receiver:  make(chan Message),
		clientMux: NewMuxAgent(),

		upgrader: websocket.Upgrader{},
	}

	go wa.handler()

	return
}

// handler is the "generic" handler for a WebSocketAgent.
func (w *WebSocketAgent) handler() {
	for msg := range w.receiver {
		w.clientMux.MessageReceiver() <- msg

		if _, isShutdown := msg.(ShutdownMessage); isShutdown {
			log.Info("WebSocketAgent received a shutdown")
			return
		}
	}
}

// ServeHTTP must be bound to a HTTP endpoint, e.g., to /ws by a http.ServeMux.
func (w *WebSocketAgent) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	conn, connErr := w.upgrader.Upgrade(rw, r, nil)
	if connErr != nil {
		log.WithError(connErr).Warn("Upgrading HTTP request to WebSocket errored")
		return
	}

	client := newWebAgentClient(conn)
	w.clientMux.Register(client)

	client.start()
}

// Endpoints of all currently connected clients.
func (w *WebSocketAgent) Endpoints() []bpv7.EndpointID {
	return w.clientMux.Endpoints()
}

// MessageReceiver is a channel on which the ApplicationAgent must listen for incoming Messages.
func (w *WebSocketAgent) MessageReceiver() chan Message {
	return w.receiver
}

// MessageSender is a channel to which the ApplicationAgent can send outgoing Messages.
func (w *WebSocketAgent) MessageSender() chan Message {
	return w.clientMux.MessageSender()
}
