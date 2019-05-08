package discovery

import (
	"reflect"
	"testing"

	"github.com/dtn7/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

func TestDiscoveryMessageCbor(t *testing.T) {
	var tests = []DiscoveryMessage{
		DiscoveryMessage{
			Type:     MTCP,
			Endpoint: bundle.MustNewEndpointID("dtn:foobar"),
			Port:     8000,
		},
		DiscoveryMessage{
			Type:        MTCP,
			Endpoint:    bundle.MustNewEndpointID("dtn:foobar"),
			Port:        8000,
			Additionals: []byte("gumo"),
		},
		DiscoveryMessage{
			Type:     TCPCLV4,
			Endpoint: bundle.MustNewEndpointID("ipn:1337.23"),
			Port:     12345,
		},
		DiscoveryMessage{
			Type:        TCPCLV4,
			Endpoint:    bundle.MustNewEndpointID("ipn:1337.23"),
			Port:        12345,
			Additionals: []byte("gumo"),
		},
	}

	for _, dmIn := range tests {
		buff, err := DiscoveryMessagesToCbor([]DiscoveryMessage{dmIn})
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Decode into another DiscoveryMessage
		dmsOut, err := NewDiscoveryMessagesFromCbor(buff)
		if err != nil {
			t.Fatalf("Decoding failed: %v", err)
		}

		if l := len(dmsOut); l != 1 {
			t.Fatalf("Length of decoded DiscoveryMessages is %d != 1", l)
		}

		if !reflect.DeepEqual(dmIn, dmsOut[0]) {
			t.Fatalf("Decoded DiscoveryMessage differs: %v became %v", dmIn, dmsOut[0])
		}

		// Decode as unknown
		var dmGeneric interface{}
		var dec = codec.NewDecoderBytes(buff, new(codec.CborHandle))

		if err := dec.Decode(&dmGeneric); err != nil {
			t.Fatalf("Decoding into an interface failed: %v", err)
		}

		if ty := reflect.TypeOf(dmGeneric); ty.Kind() != reflect.Slice {
			t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
		}

		const outerLen = 1
		const innerLen = 4
		if arr := dmGeneric.([]interface{}); len(arr) != outerLen {
			t.Errorf("CBOR-Array has wrong length: %d instead of %d",
				len(arr), outerLen)

			innerArr := arr[0].([]interface{})
			if len(innerArr) != innerLen {
				t.Errorf("CBOR-Array has wrong length: %d instead of %d",
					len(innerArr), innerLen)
			}
		}

		t.Logf("%x", buff)
	}
}
