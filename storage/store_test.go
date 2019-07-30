package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func setupStoreDir(t *testing.T) string {
	filePath, err := ioutil.TempFile("", "store")

	if err != nil {
		t.Fatal(err)
	} else {
		// We don't want this file; just its path
		os.Remove(filePath.Name())
	}

	return filePath.Name()
}

func TestStore(t *testing.T) {
	dir := setupStoreDir(t)
	defer os.RemoveAll(dir)

	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	b, bErr := bundle.Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("10m").
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	if err := store.Push(b); err != nil {
		t.Fatal(err)
	}

	if _, err := store.QueryId(b.ID()); err != nil {
		t.Fatal(err)
	}
	// TODO: fetch bundle, compare

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}
