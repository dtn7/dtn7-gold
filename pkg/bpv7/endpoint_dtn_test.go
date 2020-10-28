// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewDtnEndpoint(t *testing.T) {
	tests := []struct {
		uri       string
		nodeName  string
		demux     string
		isDtnNone bool
		valid     bool
	}{
		{"dtn:none", "", "", true, true},
		{"dtn://foo/", "foo", "", false, true},
		{"dtn://foo/bar", "foo", "bar", false, true},
		{"dtn://foo/bar/buz", "foo", "bar/buz", false, true},
		{"dtn://FOO/", "FOO", "", false, true},
		{"dtn://23/", "23", "", false, true},
		{"dtn://1a2b3c/", "1a2b3c", "", false, true},
		{"dtn://a1-b2.c3_d4/", "a1-b2.c3_d4", "", false, true},
		{"dtn:foo", "", "", false, false},     // missing slashes
		{"dtn:/foo/", "", "", false, false},   // only one leading slash
		{"dtn://foo", "", "", false, false},   // missing trailing slash
		{"dtn:///bar", "", "", false, false},  // empty node name
		{"dtn://f^oo/", "", "", false, false}, // invalid char (^) in node name
		{"dtn:", "", "", false, false},        // missing SSP
		{"dtn", "", "", false, false},         // missing SSP and ":"
		{"uff:uff", "", "", false, false},     // just no
		{"", "", "", false, false},            // nothing
	}

	for _, test := range tests {
		ep, err := NewDtnEndpoint(test.uri)

		if err == nil != test.valid {
			t.Fatalf("%s: expected valid = %t, got err: %v", test.uri, test.valid, err)
		} else if err == nil {
			ep := ep.(DtnEndpoint)

			chk := ep.NodeName == test.nodeName &&
				ep.Demux == test.demux &&
				ep.IsDtnNone == test.isDtnNone
			if !chk {
				t.Fatalf("%s: expected (%s, %s, %t), got (%s, %s, %t)", test.uri,
					test.nodeName, test.demux, test.isDtnNone, ep.NodeName, ep.Demux, ep.IsDtnNone)
			}
		}
	}
}

func TestDtnEndpointCbor(t *testing.T) {
	tests := []struct {
		ep   DtnEndpoint
		data []byte
	}{
		{DtnEndpoint{IsDtnNone: true}, []byte{0x00}},
		{DtnEndpoint{NodeName: "foo"}, []byte{0x66, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F}},
		{DtnEndpoint{NodeName: "foo", Demux: "bar"}, []byte{0x69, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F, 0x62, 0x61, 0x72}},
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
		var ep DtnEndpoint
		if err := ep.UnmarshalCbor(&buf); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ep, test.ep) {
			t.Fatalf("Expected %v, got %v", test.ep, ep)
		}
	}
}

func TestDtnEndpointUri(t *testing.T) {
	tests := []struct {
		ep        DtnEndpoint
		authority string
		path      string
	}{
		{DtnEndpoint{IsDtnNone: true}, "none", "/"},
		{DtnEndpoint{NodeName: "foobar"}, "foobar", "/"},
		{DtnEndpoint{NodeName: "foo", Demux: "bar"}, "foo", "/bar"},
		{DtnEndpoint{NodeName: "foo", Demux: "bar/"}, "foo", "/bar/"},
	}

	for _, test := range tests {
		if authority := test.ep.Authority(); test.authority != authority {
			t.Fatalf("Authority: expected %s, got %s", test.authority, authority)
		}
		if path := test.ep.Path(); test.path != path {
			t.Fatalf("Path: expected %s, got %s", test.path, path)
		}
	}
}

func TestDtnEndpointIsSingleton(t *testing.T) {
	tests := []struct {
		ep        DtnEndpoint
		singleton bool
	}{
		{DtnEndpoint{IsDtnNone: true}, false},
		{DtnEndpoint{NodeName: "foobar"}, true},
		{DtnEndpoint{NodeName: "foo", Demux: "bar"}, true},
		{DtnEndpoint{NodeName: "foo", Demux: "bar/"}, true},
		{DtnEndpoint{NodeName: "foo", Demux: "~"}, false},
		{DtnEndpoint{NodeName: "foo", Demux: "~bar"}, false},
		{DtnEndpoint{NodeName: "foo", Demux: "~bar/"}, false},
	}

	for _, test := range tests {
		if singleton := test.ep.IsSingleton(); test.singleton != singleton {
			t.Fatalf("%s: expected singleton %t, got %t", test.ep, test.singleton, singleton)
		}
	}
}
