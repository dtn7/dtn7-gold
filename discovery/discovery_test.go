package discovery

import (
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/core"
)

func TestDiscoveryMessageCbor(t *testing.T) {
	var tests = []DiscoveryMessage{
		{
			Type:     core.MTCP,
			Endpoint: bundle.MustNewEndpointID("dtn://foobar/"),
			Port:     8000,
		},
		{
			Type:     core.TCPCL,
			Endpoint: bundle.MustNewEndpointID("dtn://foobar/"),
			Port:     8000,
		},
		{
			Type:     core.MTCP,
			Endpoint: bundle.MustNewEndpointID("ipn:1337.23"),
			Port:     12345,
		},
		{
			Type:     core.TCPCL,
			Endpoint: bundle.MustNewEndpointID("ipn:1337.23"),
			Port:     12345,
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
	}
}
