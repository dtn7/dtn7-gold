// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"fmt"
	"math"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// SessInitStage models the session initialization resp. SESS_INIT exchange.
type SessInitStage struct {
	state *State

	closeChan chan struct{}
	finChan   chan struct{}
}

// Start this Stage based on the previous Stage's State.
func (ci *SessInitStage) Start(state *State) {
	ci.state = state

	ci.closeChan = make(chan struct{})
	ci.finChan = make(chan struct{})

	go ci.handle()
}

func (ci *SessInitStage) receiveMsgOrClose() (ciIn *msgs.SessionInitMessage, err error) {
	select {
	case <-ci.closeChan:
		err = StageClose
		return

	case msg := <-ci.state.MsgIn:
		var ok bool
		if ciIn, ok = msg.(*msgs.SessionInitMessage); !ok {
			err = fmt.Errorf("received message has invalid type %T", msg)
			ciIn = nil
		}
		return
	}
}

func (ci *SessInitStage) handle() {
	ciOut := msgs.NewSessionInitMessage(
		ci.state.Configuration.Keepalive,
		ci.state.Configuration.SegmentMru,
		ci.state.Configuration.TransferMru,
		ci.state.Configuration.NodeId.String())

	var (
		ciIn *msgs.SessionInitMessage
		err  error
	)

	if ci.state.Configuration.ActivePeer {
		ci.state.MsgOut <- &ciOut
		ciIn, err = ci.receiveMsgOrClose()
	} else {
		if ciIn, err = ci.receiveMsgOrClose(); err == nil {
			ci.state.MsgOut <- &ciOut
		}
	}

	if err == nil {
		ci.state.Keepalive = uint16(math.Min(float64(ci.state.Configuration.Keepalive), float64(ciIn.KeepaliveInterval)))
		ci.state.SegmentMtu = ciIn.SegmentMru
		ci.state.TransferMtu = ciIn.TransferMru
		ci.state.PeerNodeId, err = bundle.NewEndpointID(ciIn.NodeId)
	}

	ci.state.StageError = err

	close(ci.finChan)
}

// Close this Stage down.
func (ci *SessInitStage) Close() error {
	close(ci.closeChan)
	return nil
}

// Finished closes this channel to indicate this Stage has finished. Afterwards the State should be inspected.
func (ci *SessInitStage) Finished() chan struct{} {
	return ci.finChan
}
