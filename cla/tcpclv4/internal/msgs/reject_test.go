// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMessageRejectionMessage(t *testing.T) {
	tests := []struct {
		valid bool
		data  []byte
		mrm   *MessageRejectionMessage
	}{
		{true, []byte{0x06, 0x01, 0x01}, NewMessageRejectionMessage(RejectionTypeUnknown, 0x01)},
		{true, []byte{0x06, 0x03, 0x01}, NewMessageRejectionMessage(RejectionUnexpected, 0x01)},
		{false, []byte{0x07, 0x00, 0x00}, nil},
		{false, []byte{0x06, 0xF0, 0x00}, nil},
	}

	for _, test := range tests {
		var mrm = new(MessageRejectionMessage)
		var buf = bytes.NewBuffer(test.data)

		if err := mrm.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.mrm, mrm) {
			t.Fatalf("MessageRejectionMessage does not match, expected %v and got %v", test.mrm, mrm)
		}

		if err := test.mrm.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
