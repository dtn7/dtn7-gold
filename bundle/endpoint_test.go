package bundle

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestEndpointInvalid(t *testing.T) {
	testCases := []struct {
		name string
		ssp  string
	}{
		{"foo", "bar"},
	}

	for _, testCase := range testCases {
		_, err := NewEndpointID(fmt.Sprintf("%v:%v", testCase.name, testCase.ssp))
		if err == nil {
			t.Fatalf("%v:%v does not resulted in an error", testCase.name, testCase.ssp)
		}
	}
}

func TestEndpointCheckValid(t *testing.T) {
	tests := []struct {
		ep    EndpointID
		valid bool
	}{
		{EndpointID{&DtnEndpoint{IsDtnNone: true}}, true},
		{EndpointID{&IpnEndpoint{0, 0}}, false},
		{EndpointID{&IpnEndpoint{0, 1}}, false},
		{EndpointID{&IpnEndpoint{1, 0}}, false},
		{EndpointID{&IpnEndpoint{1, 1}}, true},
	}

	for _, test := range tests {
		if err := test.ep.CheckValid(); (err == nil) != test.valid {
			t.Fatalf("Endpoint ID %v resulted in error: %v", test.ep, err)
		}
	}
}

func TestEndpointCbor(t *testing.T) {
	tests := []struct {
		eid  string
		cbor []byte
	}{
		{"dtn:none", []byte{0x82, 0x01, 0x00}},
		{"dtn://foo/", []byte{0x82, 0x01, 0x66, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F}},
		{"dtn://foo/bar", []byte{0x82, 0x01, 0x69, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F, 0x62, 0x61, 0x72}},
		{"ipn:1.1", []byte{0x82, 0x02, 0x82, 0x01, 0x01}},
		{"ipn:23.42", []byte{0x82, 0x02, 0x82, 0x17, 0x18, 0x2A}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("marshal-%s", test.eid), func(t *testing.T) {
			e, _ := NewEndpointID(test.eid)

			buff := new(bytes.Buffer)
			if err := cboring.Marshal(&e, buff); err != nil {
				t.Fatalf("Marshaling %s failed: %v", test.eid, err)
			}

			if data := buff.Bytes(); !reflect.DeepEqual(data, test.cbor) {
				t.Fatalf("CBOR differs: %x != %x", data, test.cbor)
			}
		})

		t.Run(fmt.Sprintf("unmarshal-%s", test.eid), func(t *testing.T) {
			e := EndpointID{}

			buff := bytes.NewBuffer(test.cbor)
			if err := cboring.Unmarshal(&e, buff); err != nil {
				t.Fatalf("Unmarshaling %s failed: %v", test.eid, err)
			}

			if e.String() != test.eid {
				t.Fatalf("EID differs: %s != %s", e.String(), test.eid)
			}
		})
	}
}

func TestEndpointUri(t *testing.T) {
	tests := []struct {
		eid       string
		authority string
		path      string
	}{
		{"dtn:none", "none", "/"},
		{"dtn://foobar/", "foobar", "/"},
		{"dtn://foo/bar", "foo", "/bar"},
		{"dtn://foo/bar/", "foo", "/bar/"},
		{"ipn:1.1", "1", "1"},
		{"ipn:23.42", "23", "42"},
	}

	for _, test := range tests {
		ep, err := NewEndpointID(test.eid)
		if err != nil {
			t.Fatal(err)
		}

		if authority := ep.Authority(); test.authority != authority {
			t.Fatalf("Authority: expected %s, got %s", test.authority, authority)
		}
		if path := ep.Path(); test.path != path {
			t.Fatalf("Path: expected %s, got %s", test.path, path)
		}
	}
}

func TestEndpointSingleton(t *testing.T) {
	tests := []struct {
		eid       string
		singleton bool
	}{
		{"dtn:none", false},
		{"dtn://foobar/", true},
		{"dtn://foo/bar", true},
		{"dtn://foo/bar/", true},
		{"dtn://foobar/~", false},
		{"dtn://foo/~bar", false},
		{"dtn://foo/~bar/", false},
		{"ipn:1.1", true},
		{"ipn:23.42", true},
	}

	for _, test := range tests {
		ep, err := NewEndpointID(test.eid)
		if err != nil {
			t.Fatal(err)
		}

		if singleton := ep.IsSingleton(); test.singleton != singleton {
			t.Fatalf("%s: expected singleton %t, got %t", test.eid, test.singleton, singleton)
		}
	}
}
