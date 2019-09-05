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
		if data, err := test.contactHeader.MarshalBinary(); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(data, test.expectedData) {
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
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x00, 0x00}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x3F, 0x04, 0x00}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x23, 0x00}, false, ContactHeader{}},
		{[]byte{0x64, 0x74, 0x6E, 0x21, 0x04, 0x23}, false, ContactHeader{}},
	}

	for _, test := range tests {
		var ch ContactHeader
		if err := ch.UnmarshalBinary(test.data); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.contactHeader, ch) {
			t.Fatalf("ContactHeader does not match, expected %v and got %v", test.contactHeader, ch)
		}
	}
}
