// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"math"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

func TestSessInitStage(t *testing.T) {
	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	activeSessInit := &SessInitStage{}
	activeState := &State{
		Configuration: Configuration{
			ActivePeer:  true,
			Keepalive:   60,
			SegmentMru:  65535,
			TransferMru: 0xFFFFFFFF,
			NodeId:      bpv7.MustNewEndpointID("dtn://active/"),
		},
		MsgIn:  msgIn,
		MsgOut: msgOut,
	}
	activeClose := make(chan struct{})

	passiveSessInit := &SessInitStage{}
	passiveState := &State{
		Configuration: Configuration{
			ActivePeer:  false,
			Keepalive:   30,
			SegmentMru:  23,
			TransferMru: 42,
			NodeId:      bpv7.MustNewEndpointID("dtn://passive/"),
		},
		MsgIn:  msgOut,
		MsgOut: msgIn,
	}
	passiveClose := make(chan struct{})

	finChan := make(chan struct{})
	go func() { activeSessInit.Handle(activeState, activeClose); finChan <- struct{}{} }()
	go func() { passiveSessInit.Handle(passiveState, passiveClose); finChan <- struct{}{} }()

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

	keepalive := uint16(math.Min(float64(activeState.Configuration.Keepalive), float64(passiveState.Configuration.Keepalive)))
	if activeState.Keepalive != keepalive || passiveState.Keepalive != keepalive {
		t.Fatalf("expected keepalive: %d, active: %d, passive: %d", keepalive, activeState.Keepalive, passiveState.Keepalive)
	}

	if mtu := passiveState.Configuration.SegmentMru; activeState.SegmentMtu != mtu {
		t.Fatalf("active segmentu MTU %d != %d", activeState.SegmentMtu, mtu)
	}
	if mtu := activeState.Configuration.SegmentMru; passiveState.SegmentMtu != mtu {
		t.Fatalf("passive segmentu MTU %d != %d", passiveState.SegmentMtu, mtu)
	}

	if mtu := passiveState.Configuration.TransferMru; activeState.TransferMtu != mtu {
		t.Fatalf("active transfer MTU %d != %d", activeState.TransferMtu, mtu)
	}
	if mtu := activeState.Configuration.TransferMru; passiveState.TransferMtu != mtu {
		t.Fatalf("passive transfer MTU %d != %d", passiveState.TransferMtu, mtu)
	}

	if nodeId := passiveState.Configuration.NodeId; activeState.PeerNodeId != nodeId {
		t.Fatalf("active node ID %v != %v", activeState.PeerNodeId, nodeId)
	}
	if nodeId := activeState.Configuration.NodeId; passiveState.PeerNodeId != nodeId {
		t.Fatalf("passive node ID %v != %v", passiveState.PeerNodeId, nodeId)
	}
}
