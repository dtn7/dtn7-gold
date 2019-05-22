package core

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"github.com/dtn7/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// SimpleStore is an implemention of the Store interface which serializes the
// bundle packs to a file. Both Lock and Query are safed with a mutex, should
// making this Store kind of thread-safe.
type SimpleStore struct {
	bundles map[string]metaBundlePack
	mutex   sync.Mutex

	directory string
	meta      string
}

type metaBundlePack struct {
	Id          string
	Receiver    bundle.EndpointID
	Timestamp   time.Time
	Constraints map[Constraint]bool
}

func newMetaBundlePack(bp BundlePack) metaBundlePack {
	return metaBundlePack{
		Id:          bp.ID(),
		Receiver:    bp.Receiver,
		Timestamp:   bp.Timestamp,
		Constraints: bp.Constraints,
	}
}

func (mbp metaBundlePack) toBundlePack(bndl *bundle.Bundle) BundlePack {
	return BundlePack{
		Bundle:      bndl,
		Receiver:    mbp.Receiver,
		Timestamp:   mbp.Timestamp,
		Constraints: mbp.Constraints,
	}
}

// NewSimpleStore creates a SimpleStore.
func NewSimpleStore(directory string) (store *SimpleStore, err error) {
	store = &SimpleStore{
		bundles: make(map[string]metaBundlePack),

		directory: directory,
		meta:      path.Join(directory, "meta"),
	}

	if _, state := os.Stat(directory); os.IsNotExist(state) {
		err = os.Mkdir(directory, 0755)
		if err != nil {
			return
		}
	}

	if _, state := os.Stat(store.meta); !os.IsNotExist(state) {
		f, fErr := os.Open(store.meta)
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
	f, err := os.Create(store.meta)
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

func (store *SimpleStore) bundlePath(id string) string {
	f := fmt.Sprintf("%x", sha1.Sum([]byte(id)))
	return path.Join(store.directory, f)
}

func (store *SimpleStore) getBundle(id string) (bndl bundle.Bundle, err error) {
	bndlPath := store.bundlePath(id)
	if _, state := os.Stat(bndlPath); os.IsNotExist(state) {
		err = state
		return
	}

	bndlBytes, bndlBytesErr := ioutil.ReadFile(bndlPath)
	if bndlBytesErr != nil {
		err = bndlBytesErr
		return
	}

	bndl, err = bundle.NewBundleFromCbor(&bndlBytes)
	return
}

func (store *SimpleStore) setBundle(bp BundlePack) error {
	bndlPath := store.bundlePath(bp.ID())
	bndlData := bp.Bundle.ToCbor()
	return ioutil.WriteFile(bndlPath, bndlData, 0755)
}

func (store *SimpleStore) Push(bp BundlePack) error {
	isKnown := store.KnowsBundle(bp)

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.bundles[bp.ID()] = newMetaBundlePack(bp)

	var err0, err1 error
	var wg sync.WaitGroup

	if !isKnown {
		wg.Add(1)
		go func() {
			err0 = store.setBundle(bp)
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		err1 = store.sync()
		wg.Done()
	}()

	wg.Wait()

	if err0 != nil {
		return err0
	}
	return err1
}

func (store *SimpleStore) Query(sel func(BundlePack) bool) (bps []BundlePack, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for _, mbp := range store.bundles {
		bndl, bndlErr := store.getBundle(mbp.Id)
		if bndlErr != nil {
			err = bndlErr
			return
		}

		if bp := mbp.toBundlePack(&bndl); sel(bp) {
			bps = append(bps, bp)
		}
	}

	return
}

func (store *SimpleStore) KnowsBundle(bp BundlePack) bool {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for _, metaBndl := range store.bundles {
		if metaBndl.Id == bp.ID() {
			return true
		}
	}
	return false
}

func (store *SimpleStore) Close() error {
	// The SimpleStore serializes Bundles on each sync-call. This contains the
	// opening of a file, the writing and the closing. Therefore, no further
	// closing is required.
	return nil
}
