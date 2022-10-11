// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDataTransmissionMessage(t *testing.T) {
	t1data := []byte{
		// Message Header:
		0x01,
		// Message Flags, START:
		0x02,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Transfer Extension Item Length:
		0x00, 0x00, 0x00, 0x00,
		// Data Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
		// Data:
		0x75, 0x66, 0x66,
	}
	t1message := NewDataTransmissionMessage(SegmentStart, 1, []byte("uff"))

	t2data := []byte{
		// Message Header:
		0x01,
		// Message Flags, START:
		0x03,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Transfer Extension Item Length:
		0x00, 0x00, 0x00, 0x00,
		// Data Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
		// Data:
		0x75, 0x66, 0x66,
	}
	t2message := NewDataTransmissionMessage(SegmentStart|SegmentEnd, 1, []byte("uff"))

	t3data := []byte{
		// Message Header:
		0x04,
		// Message Flags, START:
		0x00,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer Extension Item Length:
		0x00, 0x00, 0x00, 0x00,
		// Data Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Data:
	}

	t4data := []byte{
		// Message Header:
		0x01,
		// Message Flags, START:
		0x00,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Transfer Extension Item Length:
		0x00, 0x00, 0x00, 0x00,
		// Data Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0F,
		// Data:
	}

	t5data := []byte{
		// Message Header:
		0x01,
		// Message Flags, START:
		0x00,
		// Transfer iD:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// Transfer Extension Item Length:
		0x00, 0x00, 0x00, 0x01,
		// Transfer Extension Items:
		0xFF,
		// Data Length:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Data:
	}
	t5message := NewDataTransmissionMessage(0, 1, nil)

	tests := []struct {
		valid     bool
		bijective bool
		data      []byte
		dtm       *DataTransmissionMessage
	}{
		{true, true, t1data, t1message},
		{true, true, t2data, t2message},
		{false, false, t3data, nil},
		{false, false, t4data, nil},
		{true, false, t5data, t5message},
	}

	for _, test := range tests {
		var dtm = new(DataTransmissionMessage)
		var buf = bytes.NewBuffer(test.data)

		if err := dtm.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.dtm, dtm) {
			t.Fatalf("DataTransmissionMessage does not match, expected %v and got %v", test.dtm, dtm)
		}

		if err := test.dtm.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); test.bijective && !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
