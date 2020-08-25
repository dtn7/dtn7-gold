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
