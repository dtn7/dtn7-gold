// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"bytes"
	"testing"
)

func TestDummyModemFragments(t *testing.T) {
	var hub = newDummyHub()
	var modems [10]Modem

	for i := 0; i < len(modems); i++ {
		modems[i] = newDummyModem(16, hub)
	}

	msg := []byte("hello world")
	_ = modems[0].Send(NewFragment(0, 0, false, false, false, msg))
	for i := 0; i < len(modems); i++ {
		if f, _ := modems[i].Receive(); !bytes.Equal(f.Payload, msg) {
			t.Fatalf("Wrong payload: expected %x, got %x", msg, f.Payload)
		}
	}
}
