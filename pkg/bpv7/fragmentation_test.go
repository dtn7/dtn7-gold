// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func TestBundleFragment(t *testing.T) {
	tests := []struct {
		payloadLen int
		mtu        int
	}{
		{1024, 256},
		{512, 256},
		{256, 256},
		{256, 128},
		{256, 96},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("payload=%d,mtu=%d", test.payloadLen, test.mtu), func(t *testing.T) {
			testBundleFragment(t, test.payloadLen, test.mtu)
		})
	}
}

func testBundleFragment(t *testing.T, payloadLen, mtu int) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, payloadLen)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(mtu)
	if err != nil {
		t.Fatal(err)
	}

	expectedOffset := uint64(0)
	for _, frag := range frags {
		if !frag.PrimaryBlock.BundleControlFlags.Has(IsFragment) {
			t.Fatal("Fragment has no Is-Fragment Bundle Control Flag")
		}

		var buff bytes.Buffer
		if err = frag.MarshalCbor(&buff); err != nil {
			t.Fatal(err)
		} else if l := buff.Len(); l > mtu {
			t.Fatalf("Fragment's length exceeds MTU, %d > %d\n%x", l, mtu, buff.Bytes())
		}

		if offset := frag.PrimaryBlock.FragmentOffset; offset != expectedOffset {
			t.Fatalf("Expected offset %d instead of %d", expectedOffset, offset)
		}

		if payloadBlock, err := frag.PayloadBlock(); err != nil {
			t.Fatal(err)
		} else {
			expectedOffset += uint64(len(payloadBlock.Value.(*PayloadBlock).Data()))
		}
	}
	if int(expectedOffset) != payloadLen {
		t.Fatalf("Final offset of %d does not equals payload length", expectedOffset)
	}
}

func TestBundleFragmentMustNotFragment(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		BundleCtrlFlags(MustNotFragmented).
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := bndl.Fragment(23); err == nil {
		t.Fatal("Bundle with Must-Not-Fragmented Bundle Control Flags did not erred")
	}
}

func TestBundleFragmentHugeMtu(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}

	if len(frags) != 1 {
		t.Fatalf("Fragmentation with huge MTU resulted in %d fragments, instead of one", len(frags))
	}
	if !reflect.DeepEqual(bndl, frags[0]) {
		t.Fatal("Bundles differ")
	}
}

func TestIsBundleReassemblable(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(128)
	if err != nil {
		t.Fatal(err)
	}

	if !IsBundleReassemblable(frags) {
		t.Fatal("Fragments are not reassemblable")
	}

	rand.Seed(23)
	rand.Shuffle(len(frags), func(i, j int) {
		frags[i], frags[j] = frags[j], frags[i]
	})

	if !IsBundleReassemblable(frags) {
		t.Fatal("Fragments are not reassemblable")
	}

	if IsBundleReassemblable(frags[:len(frags)-1]) {
		t.Fatal("Incomplete fragments are reassemblable")
	}
}

func TestReassembleFragments(t *testing.T) {
	payloadData := make([]byte, 1024)
	rand.Seed(23)
	_, _ = rand.Read(payloadData)

	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(payloadData).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(128)
	if err != nil {
		t.Fatal(err)
	}

	rand.Seed(42)
	rand.Shuffle(len(frags), func(i, j int) {
		frags[i], frags[j] = frags[j], frags[i]
	})

	bndl2, err := ReassembleFragments(frags)
	if err != nil {
		t.Fatal(err)
	}

	var buff1, buff2 bytes.Buffer
	if err = bndl.MarshalCbor(&buff1); err != nil {
		t.Fatal(err)
	}
	if err = bndl2.MarshalCbor(&buff2); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buff1.Bytes(), buff2.Bytes()) {
		t.Fatalf("Bundles differ:\n%x\n%x", buff1.Bytes(), buff2.Bytes())
	}
}

func TestReassembleFragmentsMissing(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(128)
	if err != nil {
		t.Fatal(err)
	}

	rand.Seed(42)
	rand.Shuffle(len(frags), func(i, j int) {
		frags[i], frags[j] = frags[j], frags[i]
	})

	if _, err = ReassembleFragments(frags[:len(frags)-1]); err == nil {
		t.Fatalf("Expected error for missing fragment")
	}
}
