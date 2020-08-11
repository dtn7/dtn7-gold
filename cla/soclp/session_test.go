// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"container/list"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// testSessionChannel is a helper function to sync a Session's Channel to a List, protected by a Mutex.
func testSessionChannel(c chan cla.ConvergenceStatus) (l *list.List, m *sync.Mutex) {
	l = list.New()
	m = new(sync.Mutex)

	go func() {
		for status := range c {
			m.Lock()
			l.PushBack(status)
			m.Unlock()
		}
	}()

	return
}

// testSessionChannelFind a wanted ConvergenceStatus within testSessionChannel's List.
func testSessionChannelFind(l *list.List, m *sync.Mutex, filter func(status cla.ConvergenceStatus) bool, t *testing.T) {
	m.Lock()
	defer m.Unlock()

	for fin := false; !fin; {
		if head := l.Front(); head == nil {
			t.Fatal("head is nil")
		} else if status, isStatus := head.Value.(cla.ConvergenceStatus); !isStatus {
			t.Fatalf("head is not a ConvergenceStatus, but %v (%T)", head.Value, head.Value)
		} else if filter(status) {
			fin = true
		}

		l.Remove(l.Front())
	}
}

func TestSessionSimple(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	pipe1in, pipe1out := io.Pipe()
	pipe2in, pipe2out := io.Pipe()

	session1 := &Session{
		In:          pipe1in,
		Out:         pipe2out,
		StartFunc:   nil,
		AddressFunc: func() string { return "loopback/s1" },
		Permanent:   false,
		Endpoint:    bundle.MustNewEndpointID("dtn://s1/"),
		SendTimeout: time.Second,
	}

	session2 := &Session{
		In:          pipe2in,
		Out:         pipe1out,
		StartFunc:   nil,
		AddressFunc: func() string { return "loopback/s2" },
		Permanent:   false,
		Endpoint:    bundle.MustNewEndpointID("dtn://s2/"),
		SendTimeout: time.Second,
	}

	if err, _ := session1.Start(); err != nil {
		t.Fatal(err)
	}
	if err, _ := session2.Start(); err != nil {
		t.Fatal(err)
	}

	session1Msgs, session1MsgsM := testSessionChannel(session1.Channel())
	session2Msgs, session2MsgsM := testSessionChannel(session2.Channel())

	time.Sleep(250 * time.Millisecond)
	testSessionChannelFind(session1Msgs, session1MsgsM, func(status cla.ConvergenceStatus) bool { return status.MessageType == cla.PeerAppeared }, t)
	testSessionChannelFind(session2Msgs, session2MsgsM, func(status cla.ConvergenceStatus) bool { return status.MessageType == cla.PeerAppeared }, t)

	b, bErr := bundle.Builder().
		CRC(bundle.CRC32).
		Source("dtn://s1/").
		Destination("dtn://s2/").
		CreationTimestampNow().
		Lifetime(time.Minute).
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	if err := session1.Send(&b); err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)
	testSessionChannelFind(session2Msgs, session2MsgsM, func(status cla.ConvergenceStatus) bool {
		if status.MessageType != cla.ReceivedBundle {
			return false
		}

		recBundle := status.Message.(cla.ConvergenceReceivedBundle).Bundle
		return reflect.DeepEqual(*recBundle, b)
	}, t)
}
