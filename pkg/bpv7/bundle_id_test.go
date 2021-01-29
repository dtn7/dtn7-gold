// SPDX-FileCopyrightText: 2019, 2020, 2021 Alvar Penning
// SPDX-FileCopyrightText: 2020 Claes Mogren
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestBundleIDCbor(t *testing.T) {
	tests := []struct {
		from BundleID
		to   BundleID
		l    uint64
	}{
		{
			from: BundleID{
				SourceNode: MustNewEndpointID("dtn://foo/bar"),
				Timestamp:  NewCreationTimestamp(23, 0),
				IsFragment: false,
			},
			to: BundleID{IsFragment: false},
			l:  2,
		},
		{
			from: BundleID{
				SourceNode:      MustNewEndpointID("dtn://foo/bar"),
				Timestamp:       NewCreationTimestamp(23, 0),
				IsFragment:      true,
				FragmentOffset:  23,
				TotalDataLength: 42,
			},
			to: BundleID{IsFragment: true},
			l:  4,
		},
	}

	for _, test := range tests {
		if l := test.from.Len(); l != test.l {
			t.Fatalf("Len mismatches: %d != %d", l, test.l)
		}

		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.from, buff); err != nil {
			t.Fatal(err)
		}
		if err := cboring.Unmarshal(&test.to, buff); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(test.to, test.from) {
			t.Fatalf("%v != %v", test.to, test.from)
		}
	}
}

func TestBundleIDScrub(t *testing.T) {
	tests := []struct {
		from BundleID
		to   BundleID
	}{
		{
			from: BundleID{
				SourceNode: MustNewEndpointID("dtn://foo/"),
				Timestamp:  NewCreationTimestamp(23, 42),
				IsFragment: false,
			},
			to: BundleID{
				SourceNode: MustNewEndpointID("dtn://foo/"),
				Timestamp:  NewCreationTimestamp(23, 42),
				IsFragment: false,
			},
		},
		{
			from: BundleID{
				SourceNode:      MustNewEndpointID("dtn://foo/"),
				Timestamp:       NewCreationTimestamp(23, 42),
				IsFragment:      true,
				FragmentOffset:  23,
				TotalDataLength: 42,
			},
			to: BundleID{
				SourceNode: MustNewEndpointID("dtn://foo/"),
				Timestamp:  NewCreationTimestamp(23, 42),
				IsFragment: false,
			},
		},
	}

	for _, test := range tests {
		if scrubbed := test.from.Scrub(); !reflect.DeepEqual(test.to, scrubbed) {
			t.Fatalf("Scrubbed BundleID mismatches: %v is not expected %v", test.to, scrubbed)
		}
	}
}
