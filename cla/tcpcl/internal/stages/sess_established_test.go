// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

func TestSessEstablishedStageKeepalivePingPong(t *testing.T) {
	// Channels are buffered because those are directly linked between sessions. In some cases, one session is already
	// closing down, while the other tries to send.
	msgIn := make(chan msgs.Message, 32)
	msgOut := make(chan msgs.Message, 32)

	keepaliveSec := uint16(2)

	activeSess := &SessEstablishedStage{}
	activeState := &State{
		MsgIn:     msgIn,
		MsgOut:    msgOut,
		Keepalive: keepaliveSec,
	}

	passiveSess := &SessEstablishedStage{}
	passiveState := &State{
		MsgIn:     msgOut,
		MsgOut:    msgIn,
		Keepalive: keepaliveSec,
	}

	// Start sessions
	startTime := time.Now()
	activeSess.Start(activeState)
	passiveSess.Start(passiveState)

	// Let them exchange some KEEPALIVEs
	select {
	case <-activeSess.Finished():
		t.Fatal("active finished")
	case <-passiveSess.Finished():
		t.Fatal("passive finished")
	case <-time.After(time.Duration(keepaliveSec*3) * time.Second):
	}

	// Close sessions
	_ = activeSess.Close()
	_ = passiveSess.Close()

	finChan := make(chan struct{})
	go func() { finChan <- <-activeSess.Finished() }()
	go func() { finChan <- <-passiveSess.Finished() }()

	for fins := 0; fins < 2; {
		select {
		case <-finChan:
			fins += 1
		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}

	// Check send/receive timestamps
	for _, sess := range []*SessEstablishedStage{activeSess, passiveSess} {
		if deltaSend := sess.lastSend.Sub(startTime); deltaSend < 2*time.Second {
			t.Fatalf("%v send delta is %v", sess, deltaSend)
		}
		if deltaReceive := sess.lastReceive.Sub(startTime); deltaReceive < 2*time.Second {
			t.Fatalf("%v receive delta is %v", sess, deltaReceive)
		}
	}
}

func TestSessEstablishedStageKeepaliveTimeout(t *testing.T) {
	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	keepaliveSec := uint16(2)

	sess := &SessEstablishedStage{}
	state := &State{
		MsgIn:     msgIn,
		MsgOut:    msgOut,
		Keepalive: keepaliveSec,
	}

	// Start sessions
	sess.Start(state)

	// Read outgoing KEEPALIVEs
	keepaliveCounter := int32(0)
	go func() {
		for msg := range msgOut {
			if _, ok := msg.(*msgs.KeepaliveMessage); ok {
				atomic.AddInt32(&keepaliveCounter, 1)
			}
		}
	}()

	// Wait for an error because of missing KEEPALIVEs
	select {
	case <-sess.Finished():
	case <-time.After(time.Duration(keepaliveSec*3) * time.Second):
		t.Fatal("timeout")
	}

	// Close sessions
	_ = sess.Close()

	if state.StageError == nil {
		t.Fatal("no error is stored")
	}

	// Do not check for an exact number because timing is hard. Let's go shopping.
	if atomic.LoadInt32(&keepaliveCounter) == 0 {
		t.Fatal("no KEEPALIVEs were received")
	}
}
