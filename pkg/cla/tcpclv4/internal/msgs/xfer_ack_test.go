// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDataAcknowledgementMessage(t *testing.T) {
	t1data := []byte{
		// Message Header:
		0x02,
		// Message Flags:
		0x03,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Acknowledgement Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF,
	}
	t1message := NewDataAcknowledgementMessage(SegmentEnd|SegmentStart, 1, 255)

	t2data := []byte{
		// Message Header:
		0x03,
		// Message Flags:
		0x03,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Acknowledgement Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF,
	}

	tests := []struct {
		valid bool
		data  []byte
		dam   *DataAcknowledgementMessage
	}{
		{true, t1data, t1message},
		{false, t2data, nil},
	}

	for _, test := range tests {
		var dam = new(DataAcknowledgementMessage)
		var buf = bytes.NewBuffer(test.data)

		if err := dam.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.dam, dam) {
			t.Fatalf("DataAcknowledgementMessage does not match, expected %v and got %v", test.dam, dam)
		}

		if err := test.dam.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
