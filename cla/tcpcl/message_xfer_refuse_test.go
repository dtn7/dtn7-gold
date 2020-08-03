// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTransferRefusalMessage(t *testing.T) {
	t1data := []byte{
		// Message Header:
		0x03,
		// Reason Code:
		0x00,
		// Transfer ID:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	t1message := NewTransferRefusalMessage(RefusalUnknown, 1)

	t2data := []byte{
		// Message Header:
		0x05,
		// Reason Code:
		0x00,
		// Transfer ID:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	t2message := TransferRefusalMessage{}

	t3data := []byte{
		// Message Header:
		0x03,
		// Reason Code:
		0xFF,
		// Transfer ID:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	t3message := NewTransferRefusalMessage(RefusalUnknown, 1)

	tests := []struct {
		valid bool
		data  []byte
		trm   TransferRefusalMessage
	}{
		{true, t1data, t1message},
		{false, t2data, t2message},
		{false, t3data, t3message},
	}

	for _, test := range tests {
		var trm TransferRefusalMessage
		var buf = bytes.NewBuffer(test.data)

		if err := trm.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.trm, trm) {
			t.Fatalf("TransferRefusalMessage does not match, expected %v and got %v", test.trm, trm)
		}

		if err := test.trm.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
