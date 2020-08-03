// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"fmt"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/gorilla/websocket"
)

// WebSocketAgentConnector is the client side version of the WebSocketAgent.
type WebSocketAgentConnector struct {
	conn *websocket.Conn

	msgOutChan chan webAgentMessage
	msgOutErr  chan error

	msgInBundleChan  chan bundle.Bundle
	msgInSyscallChan chan []byte

	closeSyn chan struct{}
	closeAck chan struct{}
}

// NewWebSocketAgentConnector creates a new WebSocketAgentConnector connection to a WebSocketAgent.
func NewWebSocketAgentConnector(apiUrl, endpointId string) (wac *WebSocketAgentConnector, err error) {
	var conn *websocket.Conn
	if conn, _, err = websocket.DefaultDialer.Dial(apiUrl, nil); err != nil {
		return
	}

	wac = &WebSocketAgentConnector{
		conn: conn,

		msgOutChan: make(chan webAgentMessage),
		msgOutErr:  make(chan error),

		msgInBundleChan:  make(chan bundle.Bundle),
		msgInSyscallChan: make(chan []byte),

		closeSyn: make(chan struct{}),
		closeAck: make(chan struct{}),
	}

	if err = wac.registerEndpoint(endpointId); err != nil {
		wac = nil
		return
	}

	go wac.handler()
	go wac.handleReader()

	return
}

func (wac *WebSocketAgentConnector) writeMessage(msg webAgentMessage) error {
	wc, wcErr := wac.conn.NextWriter(websocket.BinaryMessage)
	if wcErr != nil {
		return wcErr
	}

	if cborErr := marshalCbor(msg, wc); cborErr != nil {
		return cborErr
	}

	return wc.Close()
}

func (wac *WebSocketAgentConnector) readMessage() (msg webAgentMessage, err error) {
	if mt, r, rErr := wac.conn.NextReader(); rErr != nil {
		err = rErr
		return
	} else if mt != websocket.BinaryMessage {
		err = fmt.Errorf("expected binary message, got %d", mt)
		return
	} else {
		msg, err = unmarshalCbor(r)
		return
	}
}

func (wac *WebSocketAgentConnector) registerEndpoint(endpointId string) error {
	if err := wac.writeMessage(newRegisterMessage(endpointId)); err != nil {
		return err
	}

	if msg, err := wac.readMessage(); err != nil {
		return err
	} else if status, ok := msg.(*wamStatus); !ok {
		return fmt.Errorf("expected wamStatus, got %T", msg)
	} else if status.errorMsg != "" {
		return fmt.Errorf("received non-empty error message: %s", status.errorMsg)
	} else {
		return nil
	}
}

func (wac *WebSocketAgentConnector) handleReader() {
	defer close(wac.msgInBundleChan)
	defer close(wac.msgInSyscallChan)

	for {
		if msg, err := wac.readMessage(); err != nil {
			return
		} else {
			switch msg := msg.(type) {
			case *wamBundle:
				wac.msgInBundleChan <- msg.b

			case *wamSyscallResponse:
				wac.msgInSyscallChan <- msg.response

			default:
				// oof
			}
		}
	}
}

func (wac *WebSocketAgentConnector) handler() {
	defer func() {
		close(wac.closeAck)

		close(wac.msgOutChan)
		close(wac.msgOutErr)

		_ = wac.conn.Close()
	}()

	for {
		select {
		case <-wac.closeSyn:
			return

		case msg := <-wac.msgOutChan:
			wac.msgOutErr <- wac.writeMessage(msg)
		}
	}
}

// WriteBundle sends a Bundle to a server.
func (wac *WebSocketAgentConnector) WriteBundle(b bundle.Bundle) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	wac.msgOutChan <- newBundleMessage(b)
	return <-wac.msgOutErr
}

// ReadBundle returns the next incoming Bundle. This method blocks.
func (wac *WebSocketAgentConnector) ReadBundle() (b bundle.Bundle, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	b = <-wac.msgInBundleChan
	return
}

// Syscall will be send to the server. An answer or an error after a timeout will be returned.
func (wac *WebSocketAgentConnector) Syscall(request string, timeout time.Duration) (response []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	wac.msgOutChan <- newSyscallRequestMessage(request)
	if err = <-wac.msgOutErr; err != nil {
		return
	}

	select {
	case response = <-wac.msgInSyscallChan:
		return

	case <-time.After(timeout):
		err = fmt.Errorf("syscall response timed out")
		return
	}
}

// Close this WebSocketAgentConnector.
func (wac *WebSocketAgentConnector) Close() {
	defer func() {
		// channel is already closed
		_ = recover()
	}()

	close(wac.closeSyn)
	<-wac.closeAck
}
