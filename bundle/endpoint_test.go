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
		{EndpointID{&DtnEndpoint{dtnEndpointDtnNoneSsp}}, true},
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
		{"dtn:foo", []byte{0x82, 0x01, 0x63, 0x66, 0x6F, 0x6F}},
		{"dtn:foo/bar", []byte{0x82, 0x01, 0x67, 0x66, 0x6F, 0x6F, 0x2F, 0x62, 0x61, 0x72}},
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
