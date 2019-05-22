package core

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/dtn7/dtn7/bundle"
)

func testBStoreBundlePack() BundlePack {
	var bndl, _ = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn:some"), bundle.DtnNone(),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 3600),
		[]bundle.CanonicalBlock{
			bundle.NewBundleAgeBlock(1, 0, 420),
			bundle.NewPayloadBlock(0, []byte("hello world")),
		})

	bp := NewBundlePack(&bndl)
	bp.AddConstraint(ForwardPending)

	return bp
}

func testBStoreBundlePackComp(t *testing.T, bp1 BundlePack, bp2 BundlePack) {
	var comps = []struct {
		a   interface{}
		b   interface{}
		msg string
	}{
		{bp1.Bundle, bp2.Bundle, "Bundle"},
		{bp1.Receiver, bp2.Receiver, "Receiver"},
		{bp1.Timestamp.UnixNano() / 1000000, bp2.Timestamp.UnixNano() / 1000000, "Timestamp"},
		{bp1.Constraints, bp2.Constraints, "Constraints"},
	}

	for _, comp := range comps {
		if !reflect.DeepEqual(comp.a, comp.b) {
			t.Fatalf("BundlePack's %s has changed\n%v\n%v", comp.msg, comp.a, comp.b)
		}
	}
}

func testBStoreStore(t *testing.T) *BStore {
	dir, err := ioutil.TempDir("", "bstore")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewBStore(dir)
	if err != nil {
		t.Fatalf("Failed to create new BStore: %v", err)
	}

	return store
}

func testBStoreStoreClean(store *BStore, t *testing.T) {
	if err := store.Close(); err != nil {
		t.Errorf("Failed to close BStore: %v", err)
	}

	os.RemoveAll(store.dir)
}

func TestBStoreBundlePackEntry(t *testing.T) {
	bpIn := testBStoreBundlePack()

	entry, err1 := bundlePackToEntry(bpIn)
	if err1 != nil {
		t.Fatalf("Failed to encode BundlePack to Entry: %v", err1)
	}

	bpOut, err2 := entryToBundlePack(entry)
	if err2 != nil {
		t.Fatalf("Failed to decode Entry back to BundlePack: %v", err2)
	}

	testBStoreBundlePackComp(t, bpIn, bpOut)
}

func TestBStorePush(t *testing.T) {
	store := testBStoreStore(t)
	defer testBStoreStoreClean(store, t)

	bp := testBStoreBundlePack()
	for i := 0; i < 100; i++ {
		if sErr := store.Push(bp); sErr != nil {
			t.Fatalf("Failed to push BundlePack: %v", sErr)
		}
	}
}

func TestBStoreQuery(t *testing.T) {
	store := testBStoreStore(t)
	defer testBStoreStoreClean(store, t)

	// Check an empty BStore
	if l := len(QueryAll(store)); l != 0 {
		t.Fatalf("Store contains %d != 0 elements", l)
	}

	// Check one element
	bp := testBStoreBundlePack()
	store.Push(bp)

	bps := QueryAll(store)
	if l := len(bps); l != 1 {
		t.Fatalf("Store contains %d != 1 elements", l)
	}
	testBStoreBundlePackComp(t, bp, bps[0])

	// Bulk update the element
	for i := 0; i < 10; i++ {
		store.Push(bp)
	}

	if l := len(bps); l != 1 {
		t.Fatalf("Store contains %d != 1 elements after updating", l)
	}

	// Update the element
	bp.AddConstraint(ReassemblyPending)
	store.Push(bp)

	bps = QueryAll(store)
	testBStoreBundlePackComp(t, bp, bps[0])
}
