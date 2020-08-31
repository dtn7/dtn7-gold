// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

func TestStageHandlerDummy(t *testing.T) {
	s1 := dummyStage{delay: 100 * time.Millisecond}
	s2 := dummyStage{delay: 200 * time.Millisecond}
	stages := []Stage{&s1, &s2}

	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	conf := Configuration{
		ActivePeer:   true,
		ContactFlags: 0,
		Keepalive:    30,
		SegmentMru:   65535,
		TransferMru:  1048576,
		NodeId:       bundle.MustNewEndpointID("dtn://example/"),
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
	stages1 := []Stage{&ContactStage{}, &SessInitStage{}, &SessEstablishedStage{}}
	stages2 := []Stage{&ContactStage{}, &SessInitStage{}, &SessEstablishedStage{}}

	conf1 := Configuration{
		ActivePeer:  true,
		Keepalive:   10,
		SegmentMru:  1024,
		TransferMru: 1048576,
		NodeId:      bundle.MustNewEndpointID("dtn://one/"),
	}
	conf2 := Configuration{
		ActivePeer:  false,
		Keepalive:   10,
		SegmentMru:  1024,
		TransferMru: 1048576,
		NodeId:      bundle.MustNewEndpointID("dtn://two/"),
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
		if _, _, ok := sh.Exchanges(); !ok {
			t.Fatal("StageHandler is not in an exchange state")
		}

		_ = sh.Close()

		select {
		case err := <-sh.Error():
			if err != nil {
				t.Fatal(err)
			}

		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}
}
