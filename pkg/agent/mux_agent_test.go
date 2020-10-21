// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func TestMuxAgent(t *testing.T) {
	b1, err := bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://agent/mock-1/").
		CreationTimestampEpoch().
		Lifetime("24h").
		BundleAgeBlock(0).
		HopCountBlock(64).
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	mux := NewMuxAgent()

	mock1 := newMockAgent([]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://agent/mock-1/")})
	mock2 := newMockAgent([]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://agent/mock-2/")})

	mux.Register(mock1)
	mux.Register(mock2)

	mux.MessageReceiver() <- BundleMessage{b1}
	time.Sleep(500 * time.Millisecond)

	for i, mock := range []*mockAgent{mock1, mock2} {
		if msgs := mock.inbox(); len(msgs) != 1-i {
			t.Fatalf("mock agent%d did not receied %d messages; msgs := %v", i+1, 1-i, msgs)
		} else if 1-i > 0 && !reflect.DeepEqual(msgs[0].(BundleMessage).Bundle, b1) {
			t.Fatalf("message is not b1; %v %v", msgs[0], b1)
		}
	}

	mock1.MessageSender() <- ShutdownMessage{}
	time.Sleep(500 * time.Millisecond)

	select {
	case msg := <-mux.MessageSender():
		t.Fatalf("Mux forwarded shutdown message %v", msg)

	case <-time.After(250 * time.Millisecond):
		break
	}

	b1.PrimaryBlock.Destination = bpv7.MustNewEndpointID("dtn://agent/mock-2/")
	mux.MessageReceiver() <- BundleMessage{b1}
	time.Sleep(500 * time.Millisecond)

	if msgs := mock1.inbox(); len(msgs) != 0 {
		t.Fatalf("shutdowned mock agent1 received messages %v", msgs)
	}

	if msgs := mock2.inbox(); len(msgs) != 1 {
		t.Fatalf("mock agent2 did not receied messages; msgs := %v", msgs)
	} else if !reflect.DeepEqual(msgs[0].(BundleMessage).Bundle, b1) {
		t.Fatalf("message is not b1; %v %v", msgs[0], b1)
	}

	mock2.send(BundleMessage{b1})
	time.Sleep(500 * time.Millisecond)

	select {
	case msg := <-mux.MessageSender():
		if msg, ok := msg.(BundleMessage); !ok {
			t.Fatal("Message is no bundle message")
		} else if !reflect.DeepEqual(msg.Bundle, b1) {
			t.Fatalf("Expected %v, got %v", b1, msg.Bundle)
		}

	case <-time.After(250 * time.Millisecond):
		t.Fatal("Mux did not received message")
	}

	mux.MessageReceiver() <- ShutdownMessage{}
	time.Sleep(500 * time.Millisecond)

	if msgs := mock2.inbox(); len(msgs) != 1 {
		t.Fatalf("mock agent did not received one message; msgs := %v", msgs)
	} else if !reflect.DeepEqual(msgs[0], ShutdownMessage{}) {
		t.Fatalf("expected %v, got %v", ShutdownMessage{}, msgs[0])
	}
}
