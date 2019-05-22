package core

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/dtn7/dtn7/bundle"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
)

// bundlePackMeta is a struct for a BundlePack's meta data, obviously. The meta
// data and the bundle itself will be stored separatley for performance reasons.
type bundlePackMeta struct {
	Receiver    bundle.EndpointID
	Timestamp   time.Time
	Constraints map[Constraint]bool
}

func newBundlePackMeta(bp BundlePack) bundlePackMeta {
	return bundlePackMeta{
		Receiver:    bp.Receiver,
		Timestamp:   bp.Timestamp,
		Constraints: bp.Constraints,
	}
}

func (bpm bundlePackMeta) toBundlePack(bndl *bundle.Bundle) BundlePack {
	return BundlePack{
		Bundle:      bndl,
		Receiver:    bpm.Receiver,
		Timestamp:   bpm.Timestamp,
		Constraints: bpm.Constraints,
	}
}

// BStore is an implemention of a Store based on the BadgerDB.
type BStore struct {
	dir string
	db  *badger.DB
}

func NewBStore(dir string) (store *BStore, err error) {
	store = &BStore{
		dir: dir,
	}

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	opts.Logger = store

	db, dbErr := badger.Open(opts)
	if dbErr != nil {
		err = dbErr
		return
	}

	store.db = db
	return
}

func (store *BStore) Close() error {
	return store.db.Close()
}

// bundlePackToMetaEntry stores a BundlePack in a BadgerDB Entry.
func bundlePackToMetaEntry(bp BundlePack) (metaEntry badger.Entry, err error) {
	var bpRaw []byte
	var enc = codec.NewEncoderBytes(&bpRaw, new(codec.CborHandle))

	if err = enc.Encode(newBundlePackMeta(bp)); err != nil {
		return
	}

	// TODO: set ExpiresAt based on the Bundle's fields
	metaEntry = badger.Entry{
		Key:       append([]byte("m"), []byte(bp.ID())...),
		Value:     bpRaw,
		ExpiresAt: 0,
	}
	return
}

func bundlePackToBndlEntry(bp BundlePack) badger.Entry {
	// TODO: set ExpiresAt based on the Bundle's fields
	return badger.Entry{
		Key:       append([]byte("b"), []byte(bp.ID())...),
		Value:     bp.Bundle.ToCbor(),
		ExpiresAt: 0,
	}
}

func (store *BStore) Push(bp BundlePack) error {
	known := store.KnowsBundle(bp)

	me, err := bundlePackToMetaEntry(bp)
	if err != nil {
		return err
	}

	err = store.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(&me)
	})
	if err != nil {
		return err
	}

	if !known {
		be := bundlePackToBndlEntry(bp)
		return store.db.Update(func(txn *badger.Txn) error {
			return txn.SetEntry(&be)
		})
	}

	return nil
}

func (store *BStore) Query(sel func(BundlePack) bool) (bps []BundlePack, err error) {
	var meta = make(map[string]bundlePackMeta)
	var bndl = make(map[string]bundle.Bundle)

	err = store.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			if key := item.Key(); key[0] == 'm' {
				var bpm bundlePackMeta
				err = codec.NewDecoderBytes(val, new(codec.CborHandle)).Decode(&bpm)
				if err != nil {
					return err
				}
				meta[string(key[1:])] = bpm
			} else {
				b, err := bundle.NewBundleFromCbor(&val)
				if err != nil {
					return err
				}
				bndl[string(key[1:])] = b
			}
		}

		return nil
	})

	for k, m := range meta {
		b := bndl[k]
		bp := m.toBundlePack(&b)
		if sel(bp) {
			bps = append(bps, bp)
		}
	}

	return
}

func (store *BStore) KnowsBundle(bp BundlePack) bool {
	return store.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(append([]byte("m"), []byte(bp.ID())...))
		return err
	}) == nil
}

func (store *BStore) logFields() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"store": "BStore",
		"dir":   store.dir,
	})
}

func (store *BStore) Errorf(format string, args ...interface{}) {
	store.logFields().Errorf(format, args...)
}

func (store *BStore) Warningf(format string, args ...interface{}) {
	store.logFields().Warningf(format, args...)
}

func (store *BStore) Infof(format string, args ...interface{}) {
	store.logFields().Infof(format, args...)
}

func (store *BStore) Debugf(format string, args ...interface{}) {
	store.logFields().Debugf(format, args...)
}
