// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"fmt"
	"math"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// SessInitStage models the session initialization resp. SESS_INIT exchange.
type SessInitStage struct {
	state     *State
	closeChan <-chan struct{}
}

// Handle this Stage's action based on the previous Stage's State and the StageHandler's close channel.
func (ci *SessInitStage) Handle(state *State, closeChan <-chan struct{}) {
	ci.state = state
	ci.closeChan = closeChan

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
		ci.state.MsgOut <- ciOut
		ciIn, err = ci.receiveMsgOrClose()
	} else {
		ciIn, err = ci.receiveMsgOrClose()
		if err == nil {
			ci.state.MsgOut <- ciOut
		}
	}

	if err == nil {
		ci.state.Keepalive = uint16(math.Min(float64(ci.state.Configuration.Keepalive), float64(ciIn.KeepaliveInterval)))
		ci.state.SegmentMtu = ciIn.SegmentMru
		ci.state.TransferMtu = ciIn.TransferMru
		ci.state.PeerNodeId, err = bpv7.NewEndpointID(ciIn.NodeId)
	}

	ci.state.StageError = err
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
