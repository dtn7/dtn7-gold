package core

import (
	"os"
	"sync"

	"github.com/ugorji/go/codec"
)

// SimpleStore is an implemention of the Store interface which serializes the
// bundle packs to a file. Both Lock and Query are safed with a mutex, should
// making this Store kind of thread-safe.
type SimpleStore struct {
	bundles  map[string]BundlePack
	filename string
	mutex    sync.Mutex
}

// NewSimpleStore creates a SimpleStore, reading and writing data to the
// specified file.
func NewSimpleStore(filename string) (store *SimpleStore, err error) {
	store = &SimpleStore{
		bundles:  make(map[string]BundlePack),
		filename: filename,
	}

	if _, state := os.Stat(filename); !os.IsNotExist(state) {
		f, fErr := os.Open(filename)
		if fErr != nil {
			err = fErr
			return
		}

		defer f.Close()

		dec := codec.NewDecoder(f, new(codec.CborHandle))
		err = dec.Decode(&store.bundles)

	}

	return
}

func (store *SimpleStore) sync() error {
	f, err := os.Create(store.filename)
	if err != nil {
		return err
	}

	defer f.Close()

	enc := codec.NewEncoder(f, new(codec.CborHandle))
	if err := enc.Encode(store.bundles); err != nil {
		return err
	}

	return nil
}

func (store *SimpleStore) Push(bp BundlePack) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Originally, bundle packs without any constraints were removed from the
	// store, as advided in dtn-bpbis. However, removing all track of a bundle
	// whatsoever	resulted in accepting an already known bundle.
	store.bundles[bp.Bundle.ID()] = bp

	return store.sync()
}

func (store *SimpleStore) Query(sel func(BundlePack) bool) (bps []BundlePack) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for _, v := range store.bundles {
		if sel(v) {
			bps = append(bps, v)
		}
	}

	return
}
