// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

func TestContactStage(t *testing.T) {
	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	activeContact := &ContactStage{}
	activeState := &State{
		Configuration: Configuration{
			ActivePeer:   true,
			ContactFlags: msgs.ContactCanTls,
		},
		MsgIn:  msgIn,
		MsgOut: msgOut,
	}
	activeClose := make(chan struct{})

	passiveContact := &ContactStage{}
	passiveState := &State{
		Configuration: Configuration{
			ActivePeer:   false,
			ContactFlags: 0,
		},
		MsgIn:  msgOut,
		MsgOut: msgIn,
	}
	passiveClose := make(chan struct{})

	finChan := make(chan struct{})
	go func() { activeContact.Handle(activeState, activeClose); finChan <- struct{}{} }()
	go func() { passiveContact.Handle(passiveState, passiveClose); finChan <- struct{}{} }()

	for fins := 0; fins < 2; {
		select {
		case <-finChan:
			fins += 1
		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}

	if err := activeState.StageError; err != nil {
		t.Fatal(err)
	}
	if err := passiveState.StageError; err != nil {
		t.Fatal(err)
	}

	if cf := activeState.ContactFlags; cf != 0 {
		t.Fatalf("active state's contact flags are %v", cf)
	}
	if cf := passiveState.ContactFlags; cf != msgs.ContactCanTls {
		t.Fatalf("passive state's contact flags are %v", cf)
	}
}
