package core

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/ugorji/go/codec"
)

// SimpleStore is an implemention of the Store interface which serializes the
// bundle packs to a file. Both Lock and Query are safed with a mutex, should
// making this Store kind of thread-safe.
type SimpleStore struct {
	bundles map[string]metaBundlePack
	mutex   sync.Mutex
	ioMutex sync.Mutex

	directory string
	meta      string
}

type metaBundlePack struct {
	Id          string
	Receiver    bundle.EndpointID
	Timestamp   time.Time
	Constraints map[Constraint]bool
}

// hasConstraint returns true if the given constraint contains.
func (mbp metaBundlePack) hasConstraint(c Constraint) bool {
	_, ok := mbp.Constraints[c]
	return ok
}

// hasConstraints returns true if any constraint exists.
func (mbp metaBundlePack) hasConstraints() bool {
	return len(mbp.Constraints) != 0
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
	store.ioMutex.Lock()
	defer store.ioMutex.Unlock()

	bndlPath := store.bundlePath(id)
	if _, state := os.Stat(bndlPath); os.IsNotExist(state) {
		err = state
		return
	}

	bndlFile, bndlErr := os.Open(bndlPath)
	if bndlErr != nil {
		err = bndlErr
		return
	}

	err = bndl.UnmarshalCbor(bndlFile)
	return
}

func (store *SimpleStore) setBundle(bp BundlePack) error {
	store.ioMutex.Lock()
	defer store.ioMutex.Unlock()

	bndlPath := store.bundlePath(bp.ID())
	bndlFile, bndlErr := os.OpenFile(bndlPath, os.O_WRONLY|os.O_CREATE, 0755)
	if bndlErr != nil {
		return bndlErr
	}
	return bp.Bundle.MarshalCbor(bndlFile)
}

func (store *SimpleStore) delBundle(id string) error {
	store.ioMutex.Lock()
	defer store.ioMutex.Unlock()

	bndlPath := store.bundlePath(id)
	return os.Remove(bndlPath)
}

func (store *SimpleStore) Push(bp BundlePack) error {
	isKnown := store.KnowsBundle(bp)
	deletePayload := !bp.HasConstraints()

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.bundles[bp.ID()] = newMetaBundlePack(bp)

	if !isKnown {
		go store.setBundle(bp)
	} else if deletePayload {
		go store.delBundle(bp.ID())
	}

	err := store.sync()

	log.WithFields(log.Fields{
		"bundle":         bp.ID(),
		"constraints":    bp.Constraints,
		"is_known":       isKnown,
		"delete_payload": deletePayload,
	}).Debug("SimpleStore got `Push`ed")

	return err
}

func (store *SimpleStore) Query(sel func(BundlePack) bool) (bps []BundlePack, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for _, mbp := range store.bundles {
		if !mbp.hasConstraints() {
			continue
		}

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

func (store *SimpleStore) QueryId(bundleId string) (bp BundlePack, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	mbp, ok := store.bundles[bundleId]
	if !ok {
		err = fmt.Errorf("SimpleStore does not contain a bundle pack for %s", bundleId)
		return
	}

	bndl, bndlErr := store.getBundle(mbp.Id)
	if bndlErr != nil {
		err = bndlErr
		return
	}

	bp = mbp.toBundlePack(&bndl)
	return
}

func (store *SimpleStore) QueryPending() (bps []BundlePack, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for _, mbp := range store.bundles {
		chk := !mbp.hasConstraint(ReassemblyPending) &&
			(mbp.hasConstraint(ForwardPending) || mbp.hasConstraint(Contraindicated))

		log.WithFields(log.Fields{
			"bundle": mbp.Id,
			"check":  chk,
		}).Debug("QueryPending checks for bundles")

		if chk {
			bndl, bndlErr := store.getBundle(mbp.Id)
			if bndlErr != nil {
				err = bndlErr
				return
			}

			bps = append(bps, mbp.toBundlePack(&bndl))
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