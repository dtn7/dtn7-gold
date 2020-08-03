// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestWebsocketAgentMessageEnDecode(t *testing.T) {
	b, err := bundle.Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampEpoch().
		Lifetime("24h").
		BundleAgeBlock(0).
		HopCountBlock(64).
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	msgs := []webAgentMessage{
		newStatusMessage(nil),
		newStatusMessage(fmt.Errorf("oof")),
		newRegisterMessage("dtn://foobar/"),
		newBundleMessage(b),
		newSyscallRequestMessage("test"),
		newSyscallResponseMessage("foobar", []byte{0x23, 0x42, 0xAC, 0xAB}),
	}

	for _, msg := range msgs {
		var buff bytes.Buffer

		if err := marshalCbor(msg, &buff); err != nil {
			t.Fatal(err)
		}

		if msg2, err := unmarshalCbor(&buff); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(msg, msg2) {
			t.Fatalf("Messages differ: %v %v", msg, msg2)
		}
	}
}
