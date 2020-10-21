// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSessionTerminationMessage(t *testing.T) {
	t1data := []byte{0x05, 0x00, 0x00}
	t1message := NewSessionTerminationMessage(0, TerminationUnknown)

	t2data := []byte{0x05, 0x01, 0x01}
	t2message := NewSessionTerminationMessage(TerminationReply, TerminationIdleTimeout)

	t3data := []byte{0xFF, 0x00, 0x00}

	t4data := []byte{0x05, 0x00, 0xFF}

	tests := []struct {
		valid bool
		data  []byte
		stm   *SessionTerminationMessage
	}{
		{true, t1data, t1message},
		{true, t2data, t2message},
		{false, t3data, nil},
		{false, t4data, nil},
	}

	for _, test := range tests {
		var stm = new(SessionTerminationMessage)
		var buf = bytes.NewBuffer(test.data)

		if err := stm.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.stm, stm) {
			t.Fatalf("SessionTerminationMessage does not match, expected %v and got %v", test.stm, stm)
		}

		if err := test.stm.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
