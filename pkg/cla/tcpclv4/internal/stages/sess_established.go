// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"errors"
	"fmt"
	"time"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/utils"
)

var sessTermRecv = errors.New("SESS_TERM")

// SessEstablishedStage models an established TCPCLv4 session after a successfully SESS_INIT.
type SessEstablishedStage struct {
	state     *State
	closeChan <-chan struct{}

	lastReceive time.Time
	lastSend    time.Time

	keepalive *utils.KeepaliveTicker
}

// Handle this Stage's action based on the previous Stage's State and the StageHandler's close channel.
func (se *SessEstablishedStage) Handle(state *State, closeChan <-chan struct{}) {
	se.state = state
	se.closeChan = closeChan

	se.lastReceive = time.Now()
	se.lastSend = time.Now()

	se.keepalive = utils.NewKeepaliveTicker()

	if se.state.Keepalive != 0 {
		se.keepalive.Reschedule(time.Duration(se.state.Keepalive) * time.Second / 2)
	}
	defer se.keepalive.Stop()

	for {
		var err error

		select {
		case <-se.closeChan:
			err = StageClose
			_ = se.messageOut(msgs.NewSessionTerminationMessage(0, msgs.TerminationUnknown))

		case <-se.keepalive.C:
			err = se.handleKeepalive()

		case msg := <-se.state.MsgIn:
			if err = se.handleMsgIn(msg); errors.Is(err, sessTermRecv) {
				_ = se.messageOut(msgs.NewSessionTerminationMessage(msgs.TerminationReply, msgs.TerminationUnknown))
				err = StageClose
			}

		case msg := <-se.state.ExchangeMsgOut:
			err = se.messageOut(msg)
		}

		if err != nil {
			se.state.StageError = err
			return
		}
	}
}

// messageOut dispatches an outgoing message to the channel and updates the lastSend field. This method MUST be used
// instead of using the channel directly.
func (se *SessEstablishedStage) messageOut(msg msgs.Message) error {
	se.state.MsgOut <- msg
	se.lastSend = time.Now()

	return nil
}

// handleKeepalive is called from handle when the keepalive ticker ticks.
//
// This method does two things. First, it checks the last timestamp of a received message against the negotiated
// keepalive value. This errors for a stalled session. Second, the last timestamp of a sent message is also compared
// with the keepalive value. If the time delta is below 1/8 of the keepalive barrier, a KEEPALIVE message will be sent.
func (se *SessEstablishedStage) handleKeepalive() error {
	keepalive := time.Duration(se.state.Keepalive) * time.Second

	receiveDelta := time.Until(se.lastReceive.Add(keepalive))
	sendDelta := time.Until(se.lastSend.Add(keepalive))

	// Check last received message
	if receiveDelta < 0 {
		return fmt.Errorf("stalled session; last message at %v, keepalive of %v", se.lastReceive, keepalive)
	}

	// Check last send message; send a KEEPALIVE if the time delta goes to zero
	if sendDelta <= keepalive/8 {
		if err := se.messageOut(msgs.NewKeepaliveMessage()); err != nil {
			return err
		}

		se.keepalive.Reschedule(keepalive / 2)
	} else {
		se.keepalive.Reschedule(sendDelta / 2)
	}

	return nil
}

func (se *SessEstablishedStage) handleMsgIn(msg msgs.Message) (err error) {
	se.lastReceive = time.Now()

	switch msg := msg.(type) {
	case *msgs.SessionInitMessage:
		err = fmt.Errorf("unexpected SESS_INIT message")

	case *msgs.SessionTerminationMessage:
		err = sessTermRecv

	case *msgs.KeepaliveMessage:
		// nothing to do

	default:
		se.state.ExchangeMsgIn <- msg
	}
	return
}
