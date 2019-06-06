package core

import (
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestIdKeeper(t *testing.T) {
	bndl0, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
		})
	if err != nil {
		t.Errorf("Creating bundle failed: %v", err)
	}

	bndl1, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
		})
	if err != nil {
		t.Errorf("Creating bundle failed: %v", err)
	}

	var keeper = NewIdKeeper()

	keeper.update(&bndl0)
	keeper.update(&bndl1)

	if seq := bndl0.PrimaryBlock.CreationTimestamp.SequenceNumber(); seq != 0 {
		t.Errorf("First bundle's sequence number is %d", seq)
	}

	if seq := bndl1.PrimaryBlock.CreationTimestamp.SequenceNumber(); seq != 1 {
		t.Errorf("Second bundle's sequence number is %d", seq)
	}
}
