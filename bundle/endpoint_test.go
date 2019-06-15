package bundle

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestEndpointDtnNone(t *testing.T) {
	dtnNone, err := NewEndpointID("dtn:none")

	if err != nil {
		t.Errorf("dtn:none resulted in an error: %v", err)
	}

	if dtnNone.SchemeName != endpointURISchemeDTN {
		t.Errorf("dtn:none has wrong scheme name: %d", dtnNone.SchemeName)
	}
	if ty := reflect.TypeOf(dtnNone.SchemeSpecificPart); ty.Kind() != reflect.Uint64 {
		t.Errorf("dtn:none's SSP has wrong type: %T instead of uint64", ty)
	}
	if v := dtnNone.SchemeSpecificPart.(uint64); v != 0 {
		t.Errorf("dtn:none's SSP is not 0, %d", v)
	}

	if str := dtnNone.String(); str != "dtn:none" {
		t.Errorf("dtn:none's string representation is %v", str)
	}
}

func TestEndpointDtn(t *testing.T) {
	dtnEP, err := NewEndpointID("dtn:foobar")

	if err != nil {
		t.Errorf("dtn:foobar resulted in an error: %v", err)
	}

	if dtnEP.SchemeName != endpointURISchemeDTN {
		t.Errorf("dtn:foobar has wrong scheme name: %d", dtnEP.SchemeName)
	}
	if ty := reflect.TypeOf(dtnEP.SchemeSpecificPart); ty.Kind() != reflect.String {
		t.Errorf("dtn:foobar's SSP has wrong type: %T instead of string", ty)
	}
	if v := dtnEP.SchemeSpecificPart.(string); v != "foobar" {
		t.Errorf("dtn:foobar's SSP is not 'foobar', %v", v)
	}

	if str := dtnEP.String(); str != "dtn:foobar" {
		t.Errorf("dtn:foobar's string representation is %v", str)
	}
}

func TestEndpointIpn(t *testing.T) {
	ipnEP, err := NewEndpointID("ipn:23.42")

	if err != nil {
		t.Errorf("ipn:23.42 resulted in an error: %v", err)
	}

	if ipnEP.SchemeName != endpointURISchemeIPN {
		t.Errorf("ipn:23.42 has wrong scheme name: %d", ipnEP.SchemeName)
	}
	if ty := reflect.TypeOf(ipnEP.SchemeSpecificPart); ty.Kind() == reflect.Array {
		if te := ty.Elem(); te.Kind() != reflect.Uint64 {
			t.Errorf("ipn:23.42's SSP array has wrong elem-type: %T instead of uint64", te)
		}
	} else {
		t.Errorf("ipn:23.42's SSP has wrong type: %T instead of array", ty)
	}
	if v := ipnEP.SchemeSpecificPart.([2]uint64); len(v) == 2 {
		if v[0] != 23 && v[1] != 42 {
			t.Errorf("ipn:23.42's SSP != (23, 42); (%d, %d)", v[0], v[1])
		}
	} else {
		t.Errorf("ipn:23.42's SSP length is not two, %d", len(v))
	}

	if str := ipnEP.String(); str != "ipn:23.42" {
		t.Errorf("ipn:23.42's string representation is %v", str)
	}
}

func TestEndpointIpnInvalid(t *testing.T) {
	testCases := []string{
		// Wrong regular expression:
		"23.", "23", ".23", "-10.5", "10.-3", "", "foo.bar", "0x23.0x42",
		// Too small numbers
		"0.23", "23.0",
		// Too big numbers
		"23.18446744073709551616", "18446744073709551616.23",
		"23.99999999999999999999", "99999999999999999999.23",
	}

	for _, testCase := range testCases {
		_, err := NewEndpointID(fmt.Sprintf("ipn:%v", testCase))
		if err == nil {
			t.Errorf("ipn:%v does not resulted in an error", testCase)
		}
	}
}

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
			t.Errorf("%v:%v does not resulted in an error", testCase.name, testCase.ssp)
		}
	}
}

func TestEndpointCheckValid(t *testing.T) {
	tests := []struct {
		ep    EndpointID
		valid bool
	}{
		{EndpointID{SchemeName: endpointURISchemeDTN, SchemeSpecificPart: "none"}, false},
		{EndpointID{SchemeName: endpointURISchemeDTN, SchemeSpecificPart: 0}, true},
		{EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{0, 1}}, false},
		{EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{1, 0}}, false},
		{EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}}, false},
		{EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{1, 1}}, true},
		{EndpointID{SchemeName: 23, SchemeSpecificPart: 0}, false},
	}

	for _, test := range tests {
		if err := test.ep.checkValid(); (err == nil) != test.valid {
			t.Errorf("Endpoint ID %v resulted in error: %v", test.ep, err)
		}
	}
}

var endpointTests = []struct {
	eid  string
	cbor []byte
}{
	{"dtn:none", []byte{0x82, 0x01, 0x00}},
	{"dtn:foo", []byte{0x82, 0x01, 0x63, 0x66, 0x6F, 0x6F}},
	{"dtn:foo/bar", []byte{0x82, 0x01, 0x67, 0x66, 0x6F, 0x6F, 0x2F, 0x62, 0x61, 0x72}},
	{"ipn:1.1", []byte{0x82, 0x02, 0x82, 0x01, 0x01}},
	{"ipn:23.42", []byte{0x82, 0x02, 0x82, 0x17, 0x18, 0x2A}},
}

func TestEndpointCbor(t *testing.T) {
	for _, test := range endpointTests {
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
