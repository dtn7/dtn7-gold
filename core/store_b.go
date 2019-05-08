package core

import (
	"github.com/dgraph-io/badger"
	"github.com/ugorji/go/codec"
)

// BStore is an implemention of a Store based on the BadgerDB.
type BStore struct {
	dir string
	db  *badger.DB
}

func NewBStore(dir string) (store *BStore, err error) {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	db, dbErr := badger.Open(opts)
	if dbErr != nil {
		err = dbErr
		return
	}

	store = &BStore{
		dir: dir,
		db:  db,
	}
	return
}

func (store *BStore) Close() error {
	return store.db.Close()
}

// bundlePackToEntry stores a BundlePack in a BadgerDB Entry.
func bundlePackToEntry(bp BundlePack) (entry badger.Entry, err error) {
	var bpRaw []byte = make([]byte, 0, 64)
	if err = codec.NewEncoderBytes(&bpRaw, new(codec.CborHandle)).Encode(bp); err != nil {
		return
	}

	// TODO: set ExpiresAt based on the Bundle's fields
	entry = badger.Entry{
		Key:       []byte(bp.ID()),
		Value:     bpRaw,
		ExpiresAt: 0,
	}
	return
}

// entryToBundlePack extracts the wrapped BundlePack from a BadgerDB Entry.
func entryToBundlePack(entry badger.Entry) (bp BundlePack, err error) {
	err = codec.NewDecoderBytes(entry.Value, new(codec.CborHandle)).Decode(&bp)
	return
}

func (store *BStore) Push(bp BundlePack) error {
	return store.db.Update(func(txn *badger.Txn) error {
		entry, err := bundlePackToEntry(bp)
		if err != nil {
			return err
		}

		return txn.SetEntry(&entry)
	})
}

func (store *BStore) Query(sel func(BundlePack) bool) (bps []BundlePack, err error) {
	err = store.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			val, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}

			var bp BundlePack
			if err := codec.NewDecoderBytes(val, new(codec.CborHandle)).Decode(&bp); err != nil {
				return err
			}

			if sel(bp) {
				bps = append(bps, bp)
			}
		}

		return nil
	})
	return
}
