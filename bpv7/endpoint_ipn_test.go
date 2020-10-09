// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewIpnEndpoint(t *testing.T) {
	tests := []struct {
		uri     string
		node    uint64
		service uint64
		valid   bool
	}{
		{"ipn:1.1", 1, 1, true},
		{"ipn:23.42", 23, 42, true},
		{"ipn:0.1", 0, 0, false},
		{"ipn:1.0", 0, 0, false},
		{"ipn:99999999999999999999.1", 0, 0, false},
		{"ipn:11", 0, 0, false},
		{"ipn1.1", 0, 0, false},
		{"uff:1.1", 0, 0, false},
		{"", 0, 0, false},
	}

	for _, test := range tests {
		ep, err := NewIpnEndpoint(test.uri)

		if err == nil != test.valid {
			t.Fatalf("Expected valid = %t, got err: %v", test.valid, err)
		} else if err == nil {
			if ep.(IpnEndpoint).Node != test.node || ep.(IpnEndpoint).Service != test.service {
				t.Fatalf("Expected SSP (%d, %d), got (%d, %d)",
					test.node, test.service, ep.(IpnEndpoint).Node, ep.(IpnEndpoint).Service)
			}
		}
	}
}

func TestIpnEndpointCbor(t *testing.T) {
	tests := []struct {
		ep   IpnEndpoint
		data []byte
	}{
		{IpnEndpoint{1, 1}, []byte{0x82, 0x01, 0x01}},
		{IpnEndpoint{23, 42}, []byte{0x82, 0x17, 0x18, 0x2A}},
	}

	for _, test := range tests {
		var buf bytes.Buffer

		// Marshal
		if err := test.ep.MarshalCbor(&buf); err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(buf.Bytes(), test.data) {
			t.Fatalf("Expected %v, got %v", test.data, buf.Bytes())
		}

		// Unmarshal
		var ep IpnEndpoint
		if err := ep.UnmarshalCbor(&buf); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ep, test.ep) {
			t.Fatalf("Expected %v, got %v", test.ep, ep)
		}
	}
}
