// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"io"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

func TestMessageSwitchSimple(t *testing.T) {
	const keepaliveSends = 1000

	in, out := io.Pipe()
	ms := NewMessageSwitchReaderWriter(in, out)
	incoming, outgoing, errChan := ms.Exchange()

	go func() {
		for i := 0; i < keepaliveSends; i++ {
			outgoing <- msgs.NewKeepaliveMessage()
		}
	}()

	for i := 0; i < keepaliveSends; i++ {
		select {
		case err := <-errChan:
			t.Fatal(err)

		case msg := <-incoming:
			if _, ok := msg.(*msgs.KeepaliveMessage); !ok {
				t.Fatalf("msg is %T", msg)
			}

		case <-time.After(250 * time.Millisecond):
			t.Fatal("timeout")
		}
	}

	if err := ms.Close(); err != nil {
		t.Fatal(err)
	}
}
