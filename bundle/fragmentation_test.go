package bundle

import (
	"reflect"
	"testing"
)

func TestBundleFragment(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src").
		Destination("dtn://dst").
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	frags, err := bndl.Fragment(256)
	if err != nil {
		t.Fatal(err)
	}

	expectedOffset := uint64(0)
	for _, frag := range frags {
		if !frag.PrimaryBlock.BundleControlFlags.Has(IsFragment) {
			t.Fatal("Fragment has no Is-Fragment Bundle Control Flag")
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
	if expectedOffset != 1024 {
		t.Fatalf("Final offset of %d does not equals payload length", expectedOffset)
	}
}

func TestBundleFragmentMustNotFragment(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src").
		Destination("dtn://dst").
		BundleCtrlFlags(MustNotFragmented).
		CreationTimestampNow().
		Lifetime("5m").
		PayloadBlock(make([]byte, 1024)).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := bndl.Fragment(23); err == nil {
		t.Fatal("Bundle with Must-Not-Fragmented Bundle Control Flags did not errored")
	}
}

func TestBundleFragmentHugeMtu(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src").
		Destination("dtn://dst").
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
