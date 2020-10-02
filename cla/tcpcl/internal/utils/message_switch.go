// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"bufio"
	"errors"
	"io"
	"sync/atomic"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// MessageSwitch exchanges msgs.Messages from an io.Reader and io.Writer to channels. The channels can be accessed
// through the Exchange method. If either the io.Reader or the io.Writer is closeable (io.Closer), closing should be
// performed after the MessageSwitcher has finished.
type MessageSwitch struct {
	in  io.Reader
	out io.Writer

	inChan  chan msgs.Message
	outChan chan msgs.Message
	errChan chan error

	// finished is accessed by sync.atomic functions; zero means running, everything else indicates a finished state
	finished uint32
}

// NewMessageSwitch for an io.Reader and io.Writer to exchange msgs.Messages to channels.
func NewMessageSwitch(in io.Reader, out io.Writer) (ms *MessageSwitch) {
	ms = &MessageSwitch{
		in:  in,
		out: out,

		inChan:  make(chan msgs.Message, 32),
		outChan: make(chan msgs.Message, 32),
		errChan: make(chan error),

		finished: 0,
	}

	go ms.handleIn()
	go ms.handleOut()

	return
}

func (ms *MessageSwitch) handleIn() {
	in := bufio.NewReader(ms.in)

	for {
		if atomic.LoadUint32(&ms.finished) != 0 {
			return
		}

		if msg, err := msgs.ReadMessage(in); err != nil {
			if atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
				ms.errChan <- err
			}
			return
		} else {
			ms.inChan <- msg
		}
	}
}

func (ms *MessageSwitch) handleOut() {
	out := bufio.NewWriter(ms.out)

	for msg := range ms.outChan {
		if atomic.LoadUint32(&ms.finished) != 0 {
			return
		}

		if err := msg.Marshal(out); err != nil {
			if atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
				ms.errChan <- err
			}
			return
		}
		if err := out.Flush(); err != nil {
			if atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
				ms.errChan <- err
			}
			return
		}
	}
}

// Close the MessageSwitch. An error might be returned if the internal state is already finished.
func (ms *MessageSwitch) Close() (err error) {
	if !atomic.CompareAndSwapUint32(&ms.finished, 0, 1) {
		err = errors.New("MessageSwitch has already finished")
	}

	return
}

// Exchange channels to be serialized.
//
// 	* incoming is a "receive only" channel for incoming Messages.
//	* outgoing is a "send only" channel for outgoing Messages.
//	* errChan is another "receive only" channel to propagate errors. Only one error should be sent.
func (ms *MessageSwitch) Exchange() (incoming <-chan msgs.Message, outgoing chan<- msgs.Message, errChan <-chan error) {
	incoming = ms.inChan
	outgoing = ms.outChan
	errChan = ms.errChan
	return
}
