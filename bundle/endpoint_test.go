package bundle

import (
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"
)

func TestEndpointDtnNone(t *testing.T) {
	dtnNone, err := NewEndpointID("dtn", "none")

	if err != nil {
		t.Errorf("dtn:none resulted in an error: %v", err)
	}

	if dtnNone.SchemeName != URISchemeDTN {
		t.Errorf("dtn:none has wrong scheme name: %d", dtnNone.SchemeName)
	}
	if ty := reflect.TypeOf(dtnNone.SchemeSpecificPart); ty.Kind() != reflect.Uint {
		t.Errorf("dtn:none's SSP has wrong type: %T instead of uint", ty)
	}
	if v := dtnNone.SchemeSpecificPart.(uint); v != 0 {
		t.Errorf("dtn:none's SSP is not 0, %d", v)
	}

	if str := dtnNone.String(); str != "dtn:none" {
		t.Errorf("dtn:none's string representation is %v", str)
	}
}

func TestEndpointDtn(t *testing.T) {
	dtnEP, err := NewEndpointID("dtn", "foobar")

	if err != nil {
		t.Errorf("dtn:foobar resulted in an error: %v", err)
	}

	if dtnEP.SchemeName != URISchemeDTN {
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
	ipnEP, err := NewEndpointID("ipn", "23.42")

	if err != nil {
		t.Errorf("ipn:23.42 resulted in an error: %v", err)
	}

	if ipnEP.SchemeName != URISchemeIPN {
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
		_, err := NewEndpointID("ipn", testCase)
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
		_, err := NewEndpointID(testCase.name, testCase.ssp)
		if err == nil {
			t.Errorf("%v:%v does not resulted in an error", testCase.name, testCase.ssp)
		}
	}
}

func TestEndpointCborDtnNone(t *testing.T) {
	var b []byte = make([]byte, 0, 64)
	var h codec.Handle = new(codec.CborHandle)
	var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

	ep, _ := NewEndpointID("dtn", "none")

	err := enc.Encode(ep)
	if err != nil {
		t.Errorf("CBOR encoding failed: %v", err)
	}

	var dec interface{}
	err = codec.NewDecoderBytes(b, new(codec.CborHandle)).Decode(&dec)
	if err != nil {
		t.Errorf("CBOR decoding failed: %v", err)
	}

	if ty := reflect.TypeOf(dec); ty.Kind() != reflect.Slice {
		t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
	}

	var arr []interface{} = dec.([]interface{})
	if arr[0].(uint64) != 1 || arr[1].(uint64) != 0 {
		t.Errorf("Decoded CBOR values are wrong: %d instead of 1, %d instead of 0",
			arr[0].(uint64), arr[1].(uint64))
	}
}

func TestEndpointCborDtn(t *testing.T) {
	var b []byte = make([]byte, 0, 64)
	var h codec.Handle = new(codec.CborHandle)
	var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

	ep, _ := NewEndpointID("dtn", "foobar")

	err := enc.Encode(ep)
	if err != nil {
		t.Errorf("CBOR encoding failed: %v", err)
	}

	var dec interface{}
	err = codec.NewDecoderBytes(b, new(codec.CborHandle)).Decode(&dec)
	if err != nil {
		t.Errorf("CBOR decoding failed: %v", err)
	}

	if ty := reflect.TypeOf(dec); ty.Kind() != reflect.Slice {
		t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
	}

	var arr []interface{} = dec.([]interface{})
	if arr[0].(uint64) != 1 || arr[1].(string) != "foobar" {
		t.Errorf("Decoded CBOR values are wrong: %d instead of 1, %s instead of \"foobar\"",
			arr[0].(uint64), arr[1].(string))
	}
}

func TestEndpointCborIpn(t *testing.T) {
	var b []byte = make([]byte, 0, 64)
	var h codec.Handle = new(codec.CborHandle)
	var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

	ep, _ := NewEndpointID("ipn", "23.42")

	err := enc.Encode(ep)
	if err != nil {
		t.Errorf("CBOR encoding failed: %v", err)
	}

	var dec interface{}
	err = codec.NewDecoderBytes(b, new(codec.CborHandle)).Decode(&dec)
	if err != nil {
		t.Errorf("CBOR decoding failed: %v", err)
	}

	if ty := reflect.TypeOf(dec); ty.Kind() != reflect.Slice {
		t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
	}

	var arr []interface{} = dec.([]interface{})
	if arr[0].(uint64) != 2 {
		t.Errorf("Decoded CBOR values are wrong: %d instead of 2", arr[0].(uint64))
	}

	var subarr []interface{} = arr[1].([]interface{})
	if subarr[0].(uint64) != 23 || subarr[1].(uint64) != 42 {
		t.Errorf("Decoded CBOR values are wrong: %d instead of 23, %d instead of 42",
			subarr[0].(uint64), subarr[1].(uint64))
	}
}

func TestEndpointCheckValid(t *testing.T) {
	tests := []struct {
		ep    EndpointID
		valid bool
	}{
		{EndpointID{SchemeName: URISchemeDTN, SchemeSpecificPart: "none"}, false},
		{EndpointID{SchemeName: URISchemeDTN, SchemeSpecificPart: 0}, true},
		{EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{0, 1}}, false},
		{EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{1, 0}}, false},
		{EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}}, false},
		{EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{1, 1}}, true},
		{EndpointID{SchemeName: 23, SchemeSpecificPart: 0}, false},
	}

	for _, test := range tests {
		if err := test.ep.checkValid(); (err == nil) != test.valid {
			t.Errorf("Endpoint ID %v resulted in error: %v", test.ep, err)
		}
	}
}
