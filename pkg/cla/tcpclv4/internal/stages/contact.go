// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"fmt"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// ContactStage models the initial ContactHeader exchange.
type ContactStage struct {
	state     *State
	closeChan <-chan struct{}
}

// Handle this Stage's action based on the previous Stage's State and the StageHandler's close channel.
func (cs *ContactStage) Handle(state *State, closeChan <-chan struct{}) {
	cs.state = state
	cs.closeChan = closeChan

	if cs.state.Configuration.ActivePeer {
		cs.handleActive()
	} else {
		cs.handlePassive()
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
