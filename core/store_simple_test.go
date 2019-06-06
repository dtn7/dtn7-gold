package core

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestSimpleStoreSingle(t *testing.T) {
	file, err := ioutil.TempFile("", "store")
	if err != nil {
		t.Errorf("Creating tempfile failed: %v", err)
	}

	// We don't want this file; just it's filename.
	os.Remove(file.Name())

	store, err := NewSimpleStore(file.Name())
	if err != nil {
		t.Errorf("Creating SimpleStore failed: %v", err)
	}

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Errorf("Creating bundle failed: %v", err)
	}

	bp := NewBundlePack(&bndl)
	if err := store.Push(bp); err != nil {
		t.Errorf("Pushing errored :%v", err)
	}

	if _, state := os.Stat(file.Name()); os.IsNotExist(state) {
		t.Errorf("File does not exists after Push")
	}

	bp.AddConstraint(DispatchPending)
	if err := store.Push(bp); err != nil {
		t.Errorf("Pushing errored :%v", err)
	}

	if l := len(store.bundles); l != 1 {
		t.Errorf("After pushing modified bundle pack, store's length is %d", l)
	}

	bp.PurgeConstraints()
	if err := store.Push(bp); err != nil {
		t.Errorf("Pushing errored :%v", err)
	}

	if l := len(QueryAll(store)); l != 0 {
		t.Errorf("Store is not empty after pushing bundle pack without constraints")
	}

	os.RemoveAll(file.Name())
}

func TestSimpleStoreTwoStores(t *testing.T) {
	file, err := ioutil.TempFile("", "store")
	if err != nil {
		t.Errorf("Creating tempfile failed: %v", err)
	}

	// We don't want this file; just it's filename.
	os.Remove(file.Name())

	store, err := NewSimpleStore(file.Name())
	if err != nil {
		t.Errorf("Creating SimpleStore failed: %v", err)
	}

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Errorf("Creating bundle failed: %v", err)
	}

	bp := NewBundlePack(&bndl)
	bp.AddConstraint(DispatchPending)
	if err := store.Push(bp); err != nil {
		t.Errorf("Pushing errored :%v", err)
	}

	t.Log(file.Name())

	store2, err := NewSimpleStore(file.Name())
	if err != nil {
		t.Errorf("Creating second SimpleStore failed: %v", err)
	}

	// I'm totally aware, that this is not a satisfying test. However, it seems
	// like the codec-library messes up time.Time, which is used in BundlePack
	// to mark the bundle's reception/creation timestamp.
	if len(store.bundles) != len(store2.bundles) {
		t.Errorf("Two stores differs.")
	}

	os.RemoveAll(file.Name())
}
