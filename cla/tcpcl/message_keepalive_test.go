// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"bytes"
	"testing"
)

func TestKeepaliveMessage(t *testing.T) {
	tests := []struct {
		valid bool
		data  []byte
	}{
		{true, []byte{KEEPALIVE}},
		{false, []byte{0x21}},
		{false, []byte{}},
	}

	for _, test := range tests {
		var km KeepaliveMessage
		var buf = bytes.NewBuffer(test.data)

		if err := km.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		}

		var km2 = NewKeepaliveMessage()
		if err := km2.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}

	}
}
