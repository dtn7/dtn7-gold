// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSessionInitMessage(t *testing.T) {
	t1data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x00, 0x00,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// EID Length (u16):
		0x00, 0x08,
		// EID Data:
		0x64, 0x74, 0x6e, 0x3a, 0x6e, 0x6f, 0x6e, 0x65,
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x00,
	}
	t1session := NewSessionInitMessage(0, 0, 0, "dtn:none")

	t2data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x0E, 0x10,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x68,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0xFC,
		// EID Length (u16):
		0x00, 0x0D,
		// EID Data:
		0x64, 0x74, 0x6e, 0x3a, 0x2f, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x62, 0x61, 0x72,
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x00,
	}
	t2session := NewSessionInitMessage(3600, 4200, 2300, "dtn://foo/bar")

	t3data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x00, 0x01,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x68,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0xFC,
		// EID Length (u16):
		0x00, 0x00,
		// EID Data: none
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x00,
	}
	t3session := NewSessionInitMessage(1, 4200, 2300, "")

	t4data := []byte{
		// Message Header:
		0xFF,
		// Keepalive Interval (u16):
		0x00, 0x00,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// EID Length (u16):
		0x00, 0x00,
		// EID Data: none
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x00,
	}
	t4session := SessionInitMessage{}

	t5data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x00, 0x00,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// EID Length (u16):
		0x00, 0x05,
		// EID Data:
		0x64, 0x74, 0x6e, 0x3a, 0x6e, 0x6f, 0x6e, 0x65,
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x00,
	}
	t5session := SessionInitMessage{}

	t6data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x00, 0x00,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// EID Length (u16):
		0x00, 0x00,
		// EID Data: none
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x08,
		// Session Extension Items:
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	t6session := SessionInitMessage{}

	t7data := []byte{
		// Message Header:
		0x07,
		// Keepalive Interval (u16):
		0x00, 0x00,
		// Segment MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Transfer MRU (u64):
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// EID Length (u16):
		0x00, 0x00,
		// EID Data: none
		// Session Extension Item Length (u32):
		0x00, 0x00, 0x00, 0x0F,
		// Session Extension Items:
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	t7session := SessionInitMessage{}

	tests := []struct {
		valid     bool
		bijective bool
		sim       SessionInitMessage
		data      []byte
	}{
		{true, true, t1session, t1data},
		{true, true, t2session, t2data},
		{true, true, t3session, t3data},
		{false, false, t4session, t4data},
		{false, false, t5session, t5data},
		{true, false, t6session, t6data},
		{false, false, t7session, t7data},
	}

	for _, test := range tests {
		var sim SessionInitMessage
		var buf = bytes.NewBuffer(test.data)

		if err := sim.Unmarshal(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if !reflect.DeepEqual(test.sim, sim) {
			t.Fatalf("SessionInitMessage does not match, expected %v and got %v", test.sim, sim)
		}

		if err := test.sim.Marshal(buf); err != nil {
			t.Fatal(err)
		} else if data := buf.Bytes(); test.bijective && !bytes.Equal(data, test.data) {
			t.Fatalf("Data does not match, expected %x and got %x", test.data, data)
		}
	}
}
