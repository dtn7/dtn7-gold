// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func setupStoreDir(t *testing.T) string {
	filePath, err := ioutil.TempFile("", "store")

	if err != nil {
		t.Fatal(err)
	} else {
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

	b, bErr := bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampNow().
		Lifetime("10m").
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	if err := store.Push(b); err != nil {
		t.Fatal(err)
	}

	if bi, err := store.QueryId(b.ID()); err != nil {
		t.Fatal(err)
	} else {
		if l := len(bi.Parts); l != 1 {
			t.Fatalf("BundleItem was %d parts, instead of 1", l)
		}

		if b2, err := bi.Parts[0].Load(); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(b, b2) {
			t.Fatalf("Bundle changed after loading")
		}
	}

	if bip, err := store.QueryPending(); err != nil {
		t.Fatal(err)
	} else if l := len(bip); l != 0 {
		t.Fatalf("Found %d pending BundleItem, instead of 0", l)
	}

	if bi, err := store.QueryId(b.ID()); err != nil {
		t.Fatal(err)
	} else {
		bi.Pending = true
		if err := store.Update(bi); err != nil {
			t.Fatal(err)
		}
	}

	if bip, err := store.QueryPending(); err != nil {
		t.Fatal(err)
	} else if l := len(bip); l != 1 {
		t.Fatalf("Found %d pending BundleItem, instead of 1", l)
	}

	if bi, err := store.QueryId(b.ID()); err != nil {
		t.Fatal(err)
	} else {
		bi.Expires = time.Now().Add(-1 * time.Second)
		if err := store.Update(bi); err != nil {
			t.Fatal(err)
		}
	}

	store.DeleteExpired()

	if bi, err := store.QueryId(b.ID()); err == nil {
		t.Fatalf("Deleted expired BundleItem was found: %v", bi)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}
