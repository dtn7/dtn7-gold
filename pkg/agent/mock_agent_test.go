// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// mockAgent is a trivial implementation of an ApplicationAgent, only used for testing.
type mockAgent struct {
	sync.Mutex

	endpoints []bpv7.EndpointID
	receiver  chan Message
	sender    chan Message

	queue []Message
}

// newMockAgent creates a mockAgent for the given endpoints.
func newMockAgent(endpoints []bpv7.EndpointID) (m *mockAgent) {
	m = &mockAgent{
		endpoints: endpoints,
		receiver:  make(chan Message),
		sender:    make(chan Message),
	}

	go m.handle()

	return
}

func (m *mockAgent) handle() {
	for msg := range m.receiver {
		m.Lock()
		m.queue = append(m.queue, msg)
		m.Unlock()

		if _, isShutdown := msg.(ShutdownMessage); isShutdown {
			break
		}
	}
}

// inbox returns all received messages and cleans the internal message queue.
func (m *mockAgent) inbox() (msgs []Message) {
	m.Lock()
	defer m.Unlock()

	msgs = m.queue
	m.queue = nil
	return
}

// send an outgoing Message.
func (m *mockAgent) send(msg Message) {
	m.sender <- msg
}

func (m *mockAgent) Endpoints() []bpv7.EndpointID {
	return m.endpoints
}

func (m *mockAgent) MessageReceiver() chan Message {
	return m.receiver
}

func (m *mockAgent) MessageSender() chan Message {
	return m.sender
}
func TestMockAgent(t *testing.T) {
	b0, err := bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://agent/mock/").
		CreationTimestampEpoch().
		Lifetime("24h").
		BundleAgeBlock(0).
		HopCountBlock(64).
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	b1, err := bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://agent/mock/").
		CreationTimestampNow().
		Lifetime("24h").
		PayloadBlock([]byte("gumo world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	mock := newMockAgent([]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://agent/mock/")})

	mock.MessageReceiver() <- BundleMessage{b0}
	mock.MessageReceiver() <- BundleMessage{b1}

	// Give mock's handler time to process the Messages..
	time.Sleep(500 * time.Millisecond)

	if msgs := mock.inbox(); len(msgs) != 2 {
		t.Fatalf("mock agent did not receied two messages; msgs := %v", msgs)
	} else if !reflect.DeepEqual(msgs[0].(BundleMessage).Bundle, b0) {
		t.Fatalf("first message is not b0; %v %v", msgs[0], b0)
	} else if !reflect.DeepEqual(msgs[1].(BundleMessage).Bundle, b1) {
		t.Fatalf("second message is not b1; %v %v", msgs[1], b1)
	}

	mock.MessageReceiver() <- ShutdownMessage{}

	// Give mock's handler time to process the Messages..
	time.Sleep(500 * time.Millisecond)

	if msgs := mock.inbox(); len(msgs) != 1 {
		t.Fatalf("mock agent did not received one message; msgs := %v", msgs)
	} else if !reflect.DeepEqual(msgs[0], ShutdownMessage{}) {
		t.Fatalf("expected %v, got %v", ShutdownMessage{}, msgs[0])
	}
}
