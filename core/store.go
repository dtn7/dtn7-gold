package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle/arecord"
)

// Store is an interface for a bundle storage.
type Store interface {
	// Push inserts or update the given bundle pack. Each bundle pack is
	// identified by the stored bundle's combination of sender, creation timestamp
	// and optionally fragmentation.
	// An error will be returned, if something failed. This is mostly an
	// implemention specific matter.
	Push(bp BundlePack) error

	// Query returns all bundle packs satisfying the given selection function.
	// Some convenience function, starting with "Query", are existing in the
	// core package.
	Query(func(BundlePack) bool) ([]BundlePack, error)

	// QueryId returns a bundle pack for the given bundle ID.
	QueryId(bundleId string) (BundlePack, error)

	// QueryPending queries those bundle packs,which could not be delivered
	// previously, but are complete (not fragmented).
	QueryPending() ([]BundlePack, error)

	// KnowsBundle checks if an entry with the BundlePack's ID exists.
	KnowsBundle(BundlePack) bool

	// Close closes the store.
	Close() error
}

// queryErrLog is a helper for the following helper functions to inspect and log
// occuring errors.
func queryErrLog(err error, funcName string) {
	if err != nil {
		log.WithFields(log.Fields{
			"func":  funcName,
			"error": err,
		}).Warn("Querying the store returned an error")
	}
}

// QueryAll is a helper function for Stores and queries all bundle packs.
func QueryAll(store Store) []BundlePack {
	bps, err := store.Query(func(bp BundlePack) bool {
		return bp.HasConstraints()
	})

	queryErrLog(err, "QueryAll")
	return bps
}

// QueryFromStatusReport returns all (hopefully <= 1) bundles related to the
// given StatusReport.
func QueryFromStatusReport(store Store, sr arecord.StatusReport) []BundlePack {
	// TODO: fragmentation support
	bps, err := store.Query(func(bp BundlePack) bool {
		pb := bp.Bundle.PrimaryBlock
		return pb.SourceNode == sr.RefBundle.SourceNode && pb.CreationTimestamp == sr.RefBundle.Timestamp
	})

	queryErrLog(err, "QueryFromStatusReport")
	return bps
}

// NoStore is a dummy implemention of the Store interface which represents, as
// the name indicates, no store whatsoever. Push will produce log messages and
// Query has no functionality at all.
type NoStore struct{}

func (_ NoStore) Push(bp BundlePack) error {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("NoStore got pushed an update")
	return nil
}

func (_ NoStore) Query(_ func(BundlePack) bool) ([]BundlePack, error) {
	return nil, nil
}

func (_ NoStore) KnowsBundle(_ BundlePack) bool {
	return false
}

func (_ NoStore) Close() error {
	return nil
}
