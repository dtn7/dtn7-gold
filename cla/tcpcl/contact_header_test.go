// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"bytes"
	"reflect"
	"testing"
)

func TestContactHeaderMarshal(t *testing.T) {
	tests := []struct {
		contactHeader ContactHeader
		expectedData  []byte
	}{
		{NewContactHeader(0), []byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x00}},
		{NewContactHeader(ContactCanTls), []byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x01}},
	}

	for _, test := range tests {
		var buf = new(bytes.Buffer)

		if err := test.contactHeader.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.expectedData) {
			t.Fatalf("Data does not match, expected %x and got %x", test.expectedData, data)
		}
	}
}

func TestContactHeaderUnmarshal(t *testing.T) {
	tests := []struct {
		data          []byte
		valid         bool
		contactHeader ContactHeader
	}{
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x00}, true, ContactHeader{Flags: 0}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x01}, true, ContactHeader{Flags: ContactCanTls}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x3F, 0x04, 0x00}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x23, 0x00}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x23}, true, ContactHeader{Flags: 0x23}},
	}

	for _, test := range tests {
		var ch ContactHeader
		var buf = bytes.NewBuffer(test.data)

		if err := ch.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.contactHeader, ch) {
			t.Fatalf("ContactHeader does not match, expected %v and got %v", test.contactHeader, ch)
		}
	}
}
