// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func TestPingAgent(t *testing.T) {
	ping := NewPing(bpv7.MustNewEndpointID("dtn://foo/ping"))

	bndlOut, bndlOutErr := bpv7.Builder().
		Source("dtn://bar/").
		Destination("dtn://foo/ping").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock([]byte("")).
		Build()

	if bndlOutErr != nil {
		t.Fatal(bndlOutErr)
	}

	ping.receiver <- BundleMessage{bndlOut}

	select {
	case <-time.After(500 * time.Millisecond):
		t.Fatal("PingAgent did not answer after 500ms")

	case m := <-ping.sender:
		if _, ok := m.(BundleMessage); !ok {
			t.Fatalf("Incoming message is not a BundleMessage, it's a %T", m)
		}

		bndlIn := m.(BundleMessage).Bundle
		if bndlIn.PrimaryBlock.Destination != bndlOut.PrimaryBlock.SourceNode {
			t.Fatalf("Incoming Bundle's Destination %v is not outgoing Bundle's Source %v",
				bndlIn.PrimaryBlock.Destination, bndlOut.PrimaryBlock.SourceNode)
		}
	}

	ping.receiver <- ShutdownMessage{}
}
