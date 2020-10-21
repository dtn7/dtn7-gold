// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// MessageSwitchWebSocket exchanges msgs.Messages from a *websocket.Conn to channels.
type MessageSwitchWebSocket struct {
	conn        *websocket.Conn
	messageType int

	inChan  chan msgs.Message
	outChan chan msgs.Message
	errChan chan error

	// finished is accessed by sync.atomic functions; zero means running, everything else indicates a finished state
	finished uint32
}

// NewMessageSwitchWebSocket for a *websocket.Conn to exchange msgs.Messages to channels.
func NewMessageSwitchWebSocket(conn *websocket.Conn) (ms *MessageSwitchWebSocket) {
	ms = &MessageSwitchWebSocket{
		conn:        conn,
		messageType: websocket.BinaryMessage,

		inChan:  make(chan msgs.Message, 32),
		outChan: make(chan msgs.Message, 32),
		errChan: make(chan error),
	}

	go ms.handleIn()
	go ms.handleOut()

	return
}

func (ms *MessageSwitchWebSocket) sendErr(err error) {
	if atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
		ms.errChan <- err
	}
}

func (ms *MessageSwitchWebSocket) handleIn() {
	for {
		if atomic.LoadUint32(&ms.finished) != 0 {
			return
		}

		if mt, r, err := ms.conn.NextReader(); err != nil {
			ms.sendErr(err)
			return
		} else if mt != ms.messageType {
			ms.sendErr(fmt.Errorf("expected message type %d instead of %d", ms.messageType, mt))
			return
		} else if msg, err := msgs.ReadMessage(r); err != nil {
			ms.sendErr(err)
			return
		} else {
			ms.inChan <- msg
		}
	}
}

func (ms *MessageSwitchWebSocket) handleOut() {
	for msg := range ms.outChan {
		if atomic.LoadUint32(&ms.finished) != 0 {
			return
		}

		if wc, err := ms.conn.NextWriter(ms.messageType); err != nil {
			ms.sendErr(err)
			return
		} else if err := msg.Marshal(wc); err != nil {
			ms.sendErr(err)
			return
		} else if err := wc.Close(); err != nil {
			ms.sendErr(err)
			return
		}
	}
}

// Close the MessageSwitchWebSocket. An error might be returned if the internal state is already finished.
func (ms *MessageSwitchWebSocket) Close() (err error) {
	if !atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
		err = errors.New("MessageSwitchWebSocket has already finished")
	}

	return
}

// Exchange channels to be serialized.
func (ms *MessageSwitchWebSocket) Exchange() (incoming <-chan msgs.Message, outgoing chan<- msgs.Message, errChan <-chan error) {
	incoming = ms.inChan
	outgoing = ms.outChan
	errChan = ms.errChan
	return
}
