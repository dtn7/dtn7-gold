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
		{false, []byte{0x23, 0x42}},
	}

	for _, test := range tests {
		var km KeepaliveMessage

		if err := km.UnmarshalBinary(test.data); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		}

		if data, err := NewKeepaliveMessage().MarshalBinary(); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}

	}
}
