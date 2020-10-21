// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

func TestStageHandlerDummy(t *testing.T) {
	s1 := dummyStage{delay: 100 * time.Millisecond}
	s2 := dummyStage{delay: 200 * time.Millisecond}
	stages := []StageSetup{{Stage: &s1}, {Stage: &s2}}

	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	conf := Configuration{
		ActivePeer:   true,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   65535,
		TransferMru:  1048576,
		NodeId:       bpv7.MustNewEndpointID("dtn://example/"),
	}

	sh := NewStageHandler(stages, msgIn, msgOut, conf)

	select {
	case err := <-sh.Error():
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(2 * (s1.delay + s2.delay)):
		t.Fatal("timeout")
	}
}

func TestStageHandlerPingPong(t *testing.T) {
	stages1 := []StageSetup{{Stage: &ContactStage{}}, {Stage: &SessInitStage{}}, {Stage: &SessEstablishedStage{}}}
	stages2 := []StageSetup{{Stage: &ContactStage{}}, {Stage: &SessInitStage{}}, {Stage: &SessEstablishedStage{}}}

	conf1 := Configuration{
		ActivePeer:  true,
		Keepalive:   10,
		SegmentMru:  1024,
		TransferMru: 1048576,
		NodeId:      bpv7.MustNewEndpointID("dtn://one/"),
	}
	conf2 := Configuration{
		ActivePeer:  false,
		Keepalive:   10,
		SegmentMru:  1024,
		TransferMru: 1048576,
		NodeId:      bpv7.MustNewEndpointID("dtn://two/"),
	}

	// Buffer channels because those are directly linked and one peer might stop before the other one.
	msgIn := make(chan msgs.Message, 32)
	msgOut := make(chan msgs.Message, 32)

	sh1 := NewStageHandler(stages1, msgIn, msgOut, conf1)
	sh2 := NewStageHandler(stages2, msgOut, msgIn, conf2)

	select {
	case err := <-sh1.Error():
		t.Fatal(err)

	case err := <-sh2.Error():
		t.Fatal(err)

	case <-time.After(250 * time.Millisecond):
	}

	for _, sh := range []*StageHandler{sh1, sh2} {
		_ = sh.Close()

		select {
		case err := <-sh.Error():
			if err != nil && err != StageClose {
				t.Fatal(err)
			}

		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}
}

func TestStageHandlerHooks(t *testing.T) {
	var s1Pre, s1Post, s2Pre, s2Post int64

	s1 := dummyStage{delay: 100 * time.Millisecond}
	s2 := dummyStage{delay: 200 * time.Millisecond}
	stages := []StageSetup{
		{
			Stage: &s1,
			PreHook: func(_ *StageHandler, _ *State) error {
				atomic.StoreInt64(&s1Pre, time.Now().UnixNano())
				return nil
			},
			PostHook: func(_ *StageHandler, _ *State) error {
				atomic.StoreInt64(&s1Post, time.Now().UnixNano())
				return nil
			},
		},
		{
			Stage: &s2,
			PreHook: func(_ *StageHandler, _ *State) error {
				atomic.StoreInt64(&s2Pre, time.Now().UnixNano())
				return nil
			},
			PostHook: func(_ *StageHandler, _ *State) error {
				atomic.StoreInt64(&s2Post, time.Now().UnixNano())
				return nil
			},
		}}

	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	conf := Configuration{
		ActivePeer:   true,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   65535,
		TransferMru:  1048576,
		NodeId:       bpv7.MustNewEndpointID("dtn://example/"),
	}

	sh := NewStageHandler(stages, msgIn, msgOut, conf)

	select {
	case err := <-sh.Error():
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(2 * (s1.delay + s2.delay)):
		t.Fatal("timeout")
	}

	if !(s1Pre < s1Post && s1Post < s2Pre && s2Pre < s2Post) {
		t.Fatalf("hooks failed: %d, %d, %d, %d", s1Pre, s1Post, s2Pre, s2Post)
	}
}

func TestStageHandlerHooksFail(t *testing.T) {
	hookErr := errors.New("oh noes")

	s1 := dummyStage{delay: 100 * time.Millisecond}
	stages := []StageSetup{
		{
			Stage: &s1,
			PostHook: func(_ *StageHandler, _ *State) error {
				return hookErr
			},
		},
	}

	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	conf := Configuration{
		ActivePeer:   true,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   65535,
		TransferMru:  1048576,
		NodeId:       bpv7.MustNewEndpointID("dtn://example/"),
	}

	sh := NewStageHandler(stages, msgIn, msgOut, conf)

	select {
	case err := <-sh.Error():
		if !errors.Is(err, hookErr) {
			t.Fatal(err)
		}

	case <-time.After(2 * s1.delay):
		t.Fatal("timeout")
	}
}
