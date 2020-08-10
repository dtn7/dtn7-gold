// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestMessageCbor(t *testing.T) {
	tests := []struct {
		msg     MessageType
		cborMsg []byte
	}{
		{
			msg:     NewIdentityMessage(bundle.MustNewEndpointID("dtn://foo/")),
			cborMsg: []byte{0x82, 0x00, 0x82, 0x01, 0x66, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F},
		},
		{
			msg:     NewIdentityMessage(bundle.MustNewEndpointID("ipn:23.42")),
			cborMsg: []byte{0x82, 0x00, 0x82, 0x02, 0x82, 0x17, 0x18, 0x2A},
		},
		{
			msg:     NewShutdownStatusMessage(),
			cborMsg: []byte{0x82, 0x01, 0x00},
		},
		{
			msg:     NewHeartbeatStatusMessage(),
			cborMsg: []byte{0x82, 0x01, 0x01},
		},
		{
			msg:     NewTransferAckMessage(5577006791947779410),
			cborMsg: []byte{0x82, 0x03, 0x1B, 0x4D, 0x65, 0x82, 0x21, 0x07, 0xFC, 0xFD, 0x52},
		},
	}

	for _, test := range tests {
		msg1 := Message{MessageType: test.msg}

		var buff bytes.Buffer
		if err := cboring.Marshal(&msg1, &buff); err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(buff.Bytes(), test.cborMsg) {
			t.Fatalf("expected: %v, got:%v", test.cborMsg, buff.Bytes())
		}

		msg2 := Message{}
		if err := cboring.Unmarshal(&msg2, &buff); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(msg1, msg2) {
			t.Fatalf("Messages differ: %v != %v", msg1, msg2)
		}
	}
}

func TestTransferMessageCbor(t *testing.T) {
	b1, b1Err := bundle.Builder().
		CRC(bundle.CRC32).
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime(time.Minute).
		PayloadBlock([]byte("hello world")).
		Build()
	if b1Err != nil {
		t.Fatal(b1Err)
	}

	tm1, tm1Err := NewTransferMessage(b1)
	if tm1Err != nil {
		t.Fatal(tm1Err)
	}
	msg1 := Message{MessageType: tm1}

	var buff bytes.Buffer
	if err := cboring.Marshal(&msg1, &buff); err != nil {
		t.Fatal(err)
	}

	msg2 := Message{}
	if err := cboring.Unmarshal(&msg2, &buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(msg1, msg2) {
		t.Fatalf("Messages differ: %v != %v", msg1, msg2)
	}
}
