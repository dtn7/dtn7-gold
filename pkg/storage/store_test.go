// SPDX-FileCopyrightText: 2019, 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func testStore(t *testing.T, scenario func(store *Store)) {
	filePath, err := ioutil.TempFile("", "store")
	if err != nil {
		t.Fatal(err)
	} else if err = os.Remove(filePath.Name()); err != nil {
		t.Fatal(err)
	}

	dir := filePath.Name()
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	scenario(store)

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreNop(t *testing.T) {
	testStore(t, func(_ *Store) {})
}

func TestStoreBundleLife(t *testing.T) {
	testStore(t, func(store *Store) {
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
	})
}

func TestStoreFragmented(t *testing.T) {
	testStore(t, func(store *Store) {
		payloadData := make([]byte, 1024)
		rand.Seed(23)
		_, _ = rand.Read(payloadData)

		b, bErr := bpv7.Builder().
			Source("dtn://src/").
			Destination("dtn://dest/").
			CreationTimestampNow().
			Lifetime("10m").
			PayloadBlock(payloadData).
			Build()
		if bErr != nil {
			t.Fatal(bErr)
		}

		frags, err := b.Fragment(256)
		if err != nil {
			t.Fatal(err)
		}

		rand.Seed(42)
		rand.Shuffle(len(frags), func(i, j int) {
			frags[i], frags[j] = frags[j], frags[i]
		})

		for i, frag := range frags {
			if err := store.Push(frag); err != nil {
				t.Fatal(err)
			}

			if i < len(frags)-1 {
				if bi, err := store.QueryId(frag.ID()); err != nil {
					t.Fatal(err)
				} else if bi.IsComplete() {
					t.Fatal("Incomplete Bundle is marked as complete")
				}
			}
		}

		if bi, err := store.QueryId(frags[len(frags)-1].ID()); err != nil {
			t.Fatal(err)
		} else {
			if !bi.IsComplete() {
				t.Fatal("Bundle is marked as incomplete")
			} else if l := len(bi.Parts); l != len(frags) {
				t.Fatalf("BundleItem has %d parts instead of %d", l, len(frags))
			}

			b2, err := bi.Load()
			if err != nil {
				t.Fatal(err)
			}

			var buff1, buff2 bytes.Buffer
			if err = b.MarshalCbor(&buff1); err != nil {
				t.Fatal(err)
			}
			if err = b2.MarshalCbor(&buff2); err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(buff1.Bytes(), buff2.Bytes()) {
				t.Fatalf("Bundles differ:\n%x\n%x", buff1.Bytes(), buff2.Bytes())
			}
		}
	})
}
