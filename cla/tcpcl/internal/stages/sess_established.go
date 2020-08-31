// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"fmt"
	"time"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/utils"
)

// SessEstablishedStage models an established TCPCLv4 session after a successfully SESS_INIT.
type SessEstablishedStage struct {
	state *State

	closeChan chan struct{}
	finChan   chan struct{}

	xmsgOut chan msgs.Message
	xmsgIn  chan msgs.Message

	lastReceive time.Time
	lastSend    time.Time

	keepalive *utils.KeepaliveTicker
}

// Start this Stage based on the previous Stage's State.
func (se *SessEstablishedStage) Start(state *State) {
	se.state = state

	se.closeChan = make(chan struct{})
	se.finChan = make(chan struct{})

	se.xmsgOut = make(chan msgs.Message)
	se.xmsgIn = make(chan msgs.Message)

	se.lastReceive = time.Now()
	se.lastSend = time.Now()

	se.keepalive = utils.NewKeepaliveTicker()

	go se.handle()
}

func (se *SessEstablishedStage) handle() {
	defer close(se.finChan)

	if se.state.Keepalive != 0 {
		se.keepalive.Reschedule(time.Duration(se.state.Keepalive) * time.Second / 2)
	}
	defer se.keepalive.Stop()

	for {
		var err error

		select {
		case <-se.closeChan:
			err = StageClose

		case <-se.keepalive.C:
			err = se.handleKeepalive()

		case msg := <-se.state.MsgIn:
			err = se.handleMsgIn(msg)

		case msg := <-se.xmsgOut:
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

	/* TODO
	case *msgs.SessionTerminationMessage:
	*/

	case *msgs.KeepaliveMessage:
		// nothing to do

	default:
		se.xmsgIn <- msg
	}
	return
}

// Exchanges returns two optional channels for Message exchange with the peer. Those channels are only available iff
// the third exchangeOk variable is true. First channel is to send outgoing Messages to the peer, e.g.,
// XFER_SEGMENTs, XFER_ACKs, XFER_REFUSE, MSG_REFUSE, or SESS_TERM. The other channel receives incoming messages.
func (se *SessEstablishedStage) Exchanges() (outgoing chan<- msgs.Message, incoming <-chan msgs.Message, exchangeOk bool) {
	outgoing = se.xmsgOut
	incoming = se.xmsgIn
	exchangeOk = true
	return
}

// Close this Stage down.
func (se *SessEstablishedStage) Close() error {
	close(se.closeChan)
	return nil
}

// Finished closes this channel to indicate this Stage has finished. Afterwards the State should be inspected.
func (se *SessEstablishedStage) Finished() <-chan struct{} {
	return se.finChan
}
