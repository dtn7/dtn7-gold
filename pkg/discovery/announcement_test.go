// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
// SPDX-FileCopyrightText: 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package discovery

import (
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
)

func TestDiscoveryMessageCbor(t *testing.T) {
	var tests = []Announcement{
		{
			Type:     cla.MTCP,
			Endpoint: bpv7.MustNewEndpointID("dtn://foobar/"),
			Port:     8000,
		},
		{
			Type:     cla.TCPCLv4,
			Endpoint: bpv7.MustNewEndpointID("dtn://foobar/"),
			Port:     8000,
		},
		{
			Type:     cla.MTCP,
			Endpoint: bpv7.MustNewEndpointID("ipn:1337.23"),
			Port:     12345,
		},
		{
			Type:     cla.TCPCLv4,
			Endpoint: bpv7.MustNewEndpointID("ipn:1337.23"),
			Port:     12345,
		},
	}

	for _, dmIn := range tests {
		buff, err := MarshalAnnouncements([]Announcement{dmIn})
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Decode into another Announcement
		dmsOut, err := UnmarshalAnnouncements(buff)
		if err != nil {
			t.Fatalf("Decoding failed: %v", err)
		}

		if l := len(dmsOut); l != 1 {
			t.Fatalf("Length of decoded DiscoveryMessages is %d != 1", l)
		}

		if !reflect.DeepEqual(dmIn, dmsOut[0]) {
			t.Fatalf("Decoded Announcement differs: %v became %v", dmIn, dmsOut[0])
		}
	}
}
