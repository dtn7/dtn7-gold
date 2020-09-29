// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"errors"
	"reflect"
	"sync"
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

	if state.StageError == nil || state.StageError == StageClose {
		t.Fatal("no error is stored")
	}

	// Do not check for an exact number because timing is hard. Let's go shopping.
	if atomic.LoadInt32(&keepaliveCounter) == 0 {
		t.Fatal("no KEEPALIVEs were received")
	}
}

func TestSessEstablishedStageMessageExchange(t *testing.T) {
	// Channels are buffered because those are directly linked between sessions. In some cases, one session is already
	// closing down, while the other tries to send.
	msgIn := make(chan msgs.Message, 32)
	msgOut := make(chan msgs.Message, 32)

	keepaliveSec := uint16(2)

	sess1 := &SessEstablishedStage{}
	state1 := &State{
		MsgIn:     msgIn,
		MsgOut:    msgOut,
		Keepalive: keepaliveSec,
	}

	sess2 := &SessEstablishedStage{}
	state2 := &State{
		MsgIn:     msgOut,
		MsgOut:    msgIn,
		Keepalive: keepaliveSec,
	}

	xch1Msgs := []msgs.Message{
		msgs.NewDataTransmissionMessage(msgs.SegmentStart, 1, []byte("hello")),
		msgs.NewDataTransmissionMessage(0, 1, []byte(" ")),
		msgs.NewDataTransmissionMessage(msgs.SegmentEnd, 1, []byte("world")),
		msgs.NewDataAcknowledgementMessage(msgs.SegmentStart|msgs.SegmentEnd, 23, 6),
	}

	xch2Msgs := []msgs.Message{
		msgs.NewDataAcknowledgementMessage(msgs.SegmentStart, 1, 5),
		msgs.NewDataAcknowledgementMessage(0, 1, 6),
		msgs.NewDataAcknowledgementMessage(msgs.SegmentEnd, 1, 11),
		msgs.NewDataTransmissionMessage(msgs.SegmentStart|msgs.SegmentEnd, 23, []byte("foobar")),
	}

	// Start sessions
	sess1.Start(state1)
	sess2.Start(state2)

	outXch1, inXch1, _ := sess1.Exchanges()
	outXch2, inXch2, _ := sess2.Exchanges()

	// Exchange the messages
	var wg sync.WaitGroup
	wg.Add(2)
	wgFin := make(chan struct{})

	go func() {
		for i, msgOut := range xch1Msgs {
			outXch1 <- msgOut

			msgIn := <-inXch1
			if !reflect.DeepEqual(msgIn, xch2Msgs[i]) {
				t.Logf("expected %v, got %v", xch2Msgs[i], msgIn)
				panic("fatal") // t.Fatal does not work within goroutines
			}
		}
		wg.Done()
	}()

	go func() {
		for i, msgOut := range xch2Msgs {
			msgIn := <-inXch2
			if !reflect.DeepEqual(msgIn, xch1Msgs[i]) {
				t.Logf("expected %v, got %v", xch1Msgs[i], msgIn)
				panic("fatal") // t.Fatal does not work within goroutines
			}

			outXch2 <- msgOut
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(wgFin)
	}()

	select {
	case <-wgFin:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout")
	}

	// Close sessions
	_ = sess1.Close()
	_ = sess2.Close()

	finChan := make(chan struct{})
	go func() { finChan <- <-sess1.Finished() }()
	go func() { finChan <- <-sess2.Finished() }()

	for fins := 0; fins < 2; {
		select {
		case <-finChan:
			fins += 1
		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}
}

func TestSessEstablishedStageSessTerm(t *testing.T) {
	// Channels are buffered because those are directly linked between sessions. In some cases, one session is already
	// closing down, while the other tries to send.
	msgIn := make(chan msgs.Message, 32)
	msgOut := make(chan msgs.Message, 32)

	keepaliveSec := uint16(30)

	sess1 := &SessEstablishedStage{}
	state1 := &State{
		MsgIn:     msgIn,
		MsgOut:    msgOut,
		Keepalive: keepaliveSec,
	}

	sess2 := &SessEstablishedStage{}
	state2 := &State{
		MsgIn:     msgOut,
		MsgOut:    msgIn,
		Keepalive: keepaliveSec,
	}

	// Start sessions
	sess1.Start(state1)
	sess2.Start(state2)

	time.Sleep(100 * time.Millisecond)

	if err := sess1.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-sess2.Finished():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout")
	}

	if err := sess1.state.StageError; !errors.Is(err, StageClose) {
		t.Fatalf("error is %v", err)
	}
	if err := sess2.state.StageError; !errors.Is(err, StageClose) {
		t.Fatalf("error is %v", err)
	}
}
