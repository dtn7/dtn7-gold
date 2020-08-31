// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"fmt"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// ContactStage models the initial ContactHeader exchange.
type ContactStage struct {
	state *State

	closeChan chan struct{}
	finChan   chan struct{}
}

// Start this Stage based on the previous Stage's State.
func (cs *ContactStage) Start(state *State) {
	cs.state = state

	cs.closeChan = make(chan struct{})
	cs.finChan = make(chan struct{})

	go cs.handle()
}

func (cs *ContactStage) handle() {
	if cs.state.Configuration.ActivePeer {
		cs.handleActive()
	} else {
		cs.handlePassive()
	}

	close(cs.finChan)
}

func (cs *ContactStage) receiveMsgOrClose() (ch *msgs.ContactHeader, err error) {
	select {
	case <-cs.closeChan:
		err = StageClose
		return

	case msg := <-cs.state.MsgIn:
		var ok bool
		if ch, ok = msg.(*msgs.ContactHeader); !ok {
			err = fmt.Errorf("received message has invalid type %T", msg)
			ch = nil
		}

		return
	}
}

func (cs *ContactStage) handleActive() {
	cs.state.MsgOut <- msgs.NewContactHeader(cs.state.Configuration.ContactFlags)

	if ch, err := cs.receiveMsgOrClose(); err != nil {
		cs.state.StageError = err
	} else {
		cs.state.ContactFlags = ch.Flags
	}
}

func (cs *ContactStage) handlePassive() {
	if ch, err := cs.receiveMsgOrClose(); err != nil {
		cs.state.StageError = err
		return
	} else {
		cs.state.ContactFlags = ch.Flags
	}

	cs.state.MsgOut <- msgs.NewContactHeader(cs.state.Configuration.ContactFlags)
}

// Exchanges are not possible in the ContactStage.
func (cs *ContactStage) Exchanges() (outgoing chan<- msgs.Message, incoming <-chan msgs.Message, exchangeOk bool) {
	return nil, nil, false
}

// Close this Stage down.
func (cs *ContactStage) Close() error {
	close(cs.closeChan)
	return nil
}

// Finished closes this channel to indicate this Stage has finished. Afterwards the State should be inspected.
func (cs *ContactStage) Finished() <-chan struct{} {
	return cs.finChan
}
