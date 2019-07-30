package storage

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/timshannon/badgerhold"
)

const (
	dirBadger string = "db"
	dirBundle string = "bndl"
)

type Store struct {
	bh *badgerhold.Store

	badgerDir string
	bundleDir string
}

func NewStore(dir string) (s *Store, err error) {
	badgerDir := path.Join(dir, dirBadger)
	bundleDir := path.Join(dir, dirBundle)

	opts := badgerhold.DefaultOptions
	opts.Dir = badgerDir
	opts.ValueDir = badgerDir

	if dirErr := os.MkdirAll(badgerDir, 0700); dirErr != nil {
		err = dirErr
		return
	}
	if dirErr := os.MkdirAll(bundleDir, 0700); dirErr != nil {
		err = dirErr
		return
	}

	if bh, bhErr := badgerhold.Open(opts); bhErr != nil {
		err = bhErr
	} else {
		s = &Store{
			bh: bh,

			badgerDir: badgerDir,
			bundleDir: bundleDir,
		}
	}
	return
}

func (s *Store) Close() error {
	return s.bh.Close()
}

func (s *Store) Push(b bundle.Bundle) error {
	bi := NewBundleItem(b, s.bundleDir)

	if biStore, err := s.QueryId(b.ID()); err != nil {
		log.WithFields(log.Fields{
			"bundle": b.ID().String(),
		}).Info("Bundle ID is unknown, inserting BundleItem")

		return s.bh.Insert(bi.Id, bi)
	} else if bi.Fragmented {
		if !biStore.Fragmented {
			log.WithFields(log.Fields{
				"bundle": b.ID().String(),
			}).Debug("Received bundle fragment, whole bundle is already stored")
			return nil
		}

		knownFragment := false
		compPart := bi.Parts[0]
		for _, part := range biStore.Parts {
			if part.FragmentOffset == compPart.FragmentOffset &&
				part.TotalDataLength == compPart.TotalDataLength {
				knownFragment = true
				break
			}
		}

		if knownFragment {
			log.WithFields(log.Fields{
				"bundle": b.ID().String(),
			}).Debug("Received bundle fragment, which is already stored")
			return nil
		} else {
			log.WithFields(log.Fields{
				"bundle": b.ID().String(),
			}).Info("Received new bundle fragment, updating BundleItem")

			biStore.Parts = append(biStore.Parts, compPart)
			return s.bh.Update(biStore.Id, biStore)
		}
	} else {
		log.WithFields(log.Fields{
			"bundle": b.ID().String(),
		}).Debug("Bundle ID is known, ignoring push")

		return nil
	}
}

func (s *Store) QueryId(bid bundle.BundleID) (bi BundleItem, err error) {
	err = s.bh.Get(bid.Scrub().String(), &bi)
	return
}
