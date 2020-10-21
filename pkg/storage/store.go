// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/timshannon/badgerhold"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

const (
	dirBadger string = "db"
	dirBundle string = "bndl"
)

// Store implements a storage for Bundles together with meta data.
type Store struct {
	bh *badgerhold.Store

	badgerDir string
	bundleDir string
}

// NewStore creates a new Store or opens an existing Store from the given path.
func NewStore(dir string) (s *Store, err error) {
	badgerDir := path.Join(dir, dirBadger)
	bundleDir := path.Join(dir, dirBundle)

	opts := badgerhold.DefaultOptions
	opts.Dir = badgerDir
	opts.ValueDir = badgerDir
	opts.Logger = log.StandardLogger()
	opts.Options.ValueLogFileSize = 1<<28 - 1

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

// Close the Store. It must not be used afterwards.
func (s *Store) Close() error {
	return s.bh.Close()
}

// Push a new/received Bundle to the Store.
func (s *Store) Push(b bpv7.Bundle) error {
	bi := newBundleItem(b, s.bundleDir)

	if biStore, err := s.QueryId(b.ID()); err != nil {
		log.WithFields(log.Fields{
			"bundle": b.ID().String(),
		}).Info("Bundle ID is unknown, inserting BundleItem")

		if err := bi.Parts[0].storeBundle(b); err != nil {
			return err
		}

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

			if err := compPart.storeBundle(b); err != nil {
				return err
			}

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

// Update an existing BundleItem.
func (s *Store) Update(bi BundleItem) error {
	log.WithFields(log.Fields{
		"bundle": bi.Id,
	}).Debug("Store updates BundleItem")

	return s.bh.Update(bi.Id, bi)
}

// Delete a BundleItem, represented by the "scrubed" BundleID.
func (s *Store) Delete(bid bpv7.BundleID) error {
	if bi, err := s.QueryId(bid); err == nil {
		log.WithFields(log.Fields{
			"bundle": bid,
		}).Info("Store deletes BundleItem")

		for _, bp := range bi.Parts {
			if err := bp.deleteBundle(); err != nil {
				log.WithFields(log.Fields{
					"bundle": bid,
					"file":   bp.Filename,
					"error":  err,
				}).Warn("Failed to delete BundlePart")
			}
		}

		return s.bh.Delete(bi.Id, BundleItem{})
	}

	return nil
}

// DeleteExpired removes all expired Bundles.
func (s *Store) DeleteExpired() {
	var bis []BundleItem
	if err := s.bh.Find(&bis, badgerhold.Where("Expires").Lt(time.Now())); err != nil {
		log.WithError(err).Warn("Failed to get expired Bundles")
		return
	}

	for _, bi := range bis {
		logger := log.WithField("bundle", bi.Id)
		if err := s.Delete(bi.BId); err != nil {
			logger.WithError(err).Warn("Failed to delete expired Bundle")
		} else {
			logger.Info("Deleted expired Bundle")
		}
	}
}

// QueryId fetches the BundleItem for the requested BundleID.
func (s *Store) QueryId(bid bpv7.BundleID) (bi BundleItem, err error) {
	err = s.bh.Get(bid.Scrub().String(), &bi)
	return
}

// QueryPending fetches all pending Bundles.
func (s *Store) QueryPending() (bis []BundleItem, err error) {
	err = s.bh.Find(&bis, badgerhold.Where("Pending").Eq(true))
	return
}

// KnowsBundle checks if such a Bundle is known.
func (s *Store) KnowsBundle(bid bpv7.BundleID) bool {
	_, err := s.QueryId(bid)
	return err != badgerhold.ErrNotFound
}
