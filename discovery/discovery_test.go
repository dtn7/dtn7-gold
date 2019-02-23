package discovery

import (
	"net"
	"reflect"
	"testing"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

func TestDiscoveryMessageCbor(t *testing.T) {
	var tests = []DiscoveryMessage{
		DiscoveryMessage{
			Type:     STCP,
			Endpoint: bundle.MustNewEndpointID("dtn:foobar"),
			Address:  net.ParseIP("172.23.23.23"),
			Port:     8000,
		},
		DiscoveryMessage{
			Type:        STCP,
			Endpoint:    bundle.MustNewEndpointID("dtn:foobar"),
			Address:     net.ParseIP("172.23.23.23"),
			Port:        8000,
			Additionals: []byte("gumo"),
		},
		DiscoveryMessage{
			Type:     TCPCLV4,
			Endpoint: bundle.MustNewEndpointID("ipn:1337.23"),
			Address:  net.ParseIP("2a01:4a0:2002:2417::2"),
			Port:     12345,
		},
		DiscoveryMessage{
			Type:        TCPCLV4,
			Endpoint:    bundle.MustNewEndpointID("ipn:1337.23"),
			Address:     net.ParseIP("2a01:4a0:2002:2417::2"),
			Port:        12345,
			Additionals: []byte("gumo"),
		},
	}

	for _, dmIn := range tests {
		buff, err := dmIn.Cbor()
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Decode into another DiscoveryMessage
		dmOut, err := NewDiscoveryMessageFromCbor(buff)
		if err != nil {
			t.Fatalf("Decoding failed: %v", err)
		}

		if !reflect.DeepEqual(dmIn, dmOut) {
			t.Fatalf("Decoded DiscoveryMessage differs: %v became %v", dmIn, dmOut)
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

		if arr := dmGeneric.([]interface{}); len(arr) != 5 {
			t.Errorf("CBOR-Array has wrong length: %d instead of %d", len(arr), 5)
		}

		t.Logf("%x", buff)
	}
}
