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

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
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
	activeClose := make(chan struct{})

	passiveSess := &SessEstablishedStage{}
	passiveState := &State{
		MsgIn:     msgOut,
		MsgOut:    msgIn,
		Keepalive: keepaliveSec,
	}
	passiveClose := make(chan struct{})

	// Start sessions
	startTime := time.Now()

	finChan := make(chan struct{})
	go func() { activeSess.Handle(activeState, activeClose); finChan <- struct{}{} }()
	go func() { passiveSess.Handle(passiveState, passiveClose); finChan <- struct{}{} }()

	// Let them exchange some KEEPALIVEs
	select {
	case <-finChan:
		t.Fatal("session finished")
	case <-time.After(time.Duration(keepaliveSec*3) * time.Second):
	}

	// Close sessions
	close(activeClose)
	close(passiveClose)

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
	closer := make(chan struct{})

	// Start sessions
	finChan := make(chan struct{})
	go func() { sess.Handle(state, closer); close(finChan) }()

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
	case <-finChan:
	case <-time.After(time.Duration(keepaliveSec*3) * time.Second):
		t.Fatal("timeout")
	}

	// Close sessions
	close(closer)

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
	exchangeMsgIn := make(chan msgs.Message, 32)
	exchangeMsgOut := make(chan msgs.Message, 32)

	keepaliveSec := uint16(2)

	sess1 := &SessEstablishedStage{}
	state1 := &State{
		MsgIn:          msgIn,
		MsgOut:         msgOut,
		ExchangeMsgIn:  exchangeMsgIn,
		ExchangeMsgOut: exchangeMsgOut,
		Keepalive:      keepaliveSec,
	}
	close1 := make(chan struct{})

	sess2 := &SessEstablishedStage{}
	state2 := &State{
		MsgIn:          msgOut,
		MsgOut:         msgIn,
		ExchangeMsgIn:  exchangeMsgOut,
		ExchangeMsgOut: exchangeMsgIn,
		Keepalive:      keepaliveSec,
	}
	close2 := make(chan struct{})

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
	finChan := make(chan struct{})
	go func() { sess1.Handle(state1, close1); finChan <- struct{}{} }()
	go func() { sess2.Handle(state2, close2); finChan <- struct{}{} }()

	outXch1, inXch1 := exchangeMsgOut, exchangeMsgIn
	outXch2, inXch2 := exchangeMsgIn, exchangeMsgOut

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
	close(close1)
	close(close2)

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
	close1 := make(chan struct{})

	sess2 := &SessEstablishedStage{}
	state2 := &State{
		MsgIn:     msgOut,
		MsgOut:    msgIn,
		Keepalive: keepaliveSec,
	}
	close2 := make(chan struct{})

	// Start sessions
	finChan := make(chan struct{})
	go func() { sess1.Handle(state1, close1); finChan <- struct{}{} }()
	go func() { sess2.Handle(state2, close2); finChan <- struct{}{} }()

	time.Sleep(100 * time.Millisecond)
	close(close1)

	for i := 0; i < 2; i++ {
		select {
		case <-finChan:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}
	}

	if err := sess1.state.StageError; !errors.Is(err, StageClose) {
		t.Fatalf("error is %v", err)
	}
	if err := sess2.state.StageError; !errors.Is(err, StageClose) {
		t.Fatalf("error is %v", err)
	}
}
