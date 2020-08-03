// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		valid    bool
		typeCode uint8
		msgType  Message
	}{
		{true, SESS_INIT, &SessionInitMessage{}},
		{true, KEEPALIVE, &KeepaliveMessage{}},
		{true, XFER_SEGMENT, &DataTransmissionMessage{}},
		{false, 0xFF, nil},
	}

	for _, test := range tests {
		if msg, err := NewMessage(test.typeCode); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if msgType := reflect.TypeOf(msg); msgType != reflect.TypeOf(test.msgType) {
			t.Fatalf("Message Type is wrong; expected := %v, got := %v", test.msgType, msgType)
		}
	}
}

func TestReadMessage(t *testing.T) {
	// cla/tcpcl/message_sess_init_test.go
	t1data := []byte{
		0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x64, 0x74, 0x6e,
		0x3a, 0x6e, 0x6f, 0x6e, 0x65, 0x00, 0x00, 0x00, 0x00,
	}
	t1msg := NewSessionInitMessage(0, 0, 0, "dtn:none")

	t2data := []byte{0x04}
	t2msg := NewKeepaliveMessage()

	// cla/tcpcl/message_xfer_segment_test.go
	t3data := []byte{
		0x01, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x75, 0x66,
		0x66,
	}
	t3msg := NewDataTransmissionMessage(SegmentStart|SegmentEnd, 1, []byte("uff"))

	// cla/tcpcl/message_xfer_segment_test.go
	t4data := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0F,
	}

	t5data := []byte{0xC0, 0xFF, 0xEE}

	tests := []struct {
		valid bool
		data  []byte
		msg   Message
	}{
		{true, t1data, &t1msg},
		{true, t2data, &t2msg},
		{true, t3data, &t3msg},
		{false, t4data, nil},
		{false, t5data, nil},
	}

	for _, test := range tests {
		var buf = bytes.NewBuffer(test.data)
		if msg, err := ReadMessage(buf); (err == nil) != test.valid {
			t.Fatalf("Error state was not expected; valid := %t, got := %v", test.valid, err)
		} else if !test.valid {
			continue
		} else if reflect.DeepEqual(msg, &test.msg) {
			t.Fatalf("Message does not match; expected := %v, got := %v", test.msg, msg)
		}
	}
}
