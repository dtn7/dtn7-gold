package core

import "log"

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
	Query(func(BundlePack) bool) []BundlePack
}

// QueryAll is a helper function for Stores and queries all bundle packs.
func QueryAll(store Store) []BundlePack {
	return store.Query(func(_ BundlePack) bool {
		return true
	})
}

// QueryFromStatusReport returns all (hopefully <= 1) bundles related to the
// given StatusReport.
func QueryFromStatusReport(store Store, sr StatusReport) []BundlePack {
	return store.Query(func(bp BundlePack) bool {
		pb := bp.Bundle.PrimaryBlock
		return pb.SourceNode == sr.SourceNode && pb.CreationTimestamp == sr.Timestamp
	})
}

// QueryPending is a helper function for Stores and queries those bundle packs,
// which could not be delivered previously, but are complete (not fragmented).
func QueryPending(store Store) []BundlePack {
	return store.Query(func(bp BundlePack) bool {
		return !bp.HasConstraint(ReassemblyPending) && bp.HasConstraint(Contraindicated)
	})
}

// KnowsBundle returns true if the requested store knows a BundlePack which
// bundle equals the requested BundlePack's.
func KnowsBundle(store Store, requested BundlePack) bool {
	return store.Query(func(bp BundlePack) bool {
		return bp.Bundle.ID() == requested.Bundle.ID()
	}) != nil
}

// NoStore is a dummy implemention of the Store interface which represents, as
// the name indicates, no store whatsoever. Push will produce log messages and
// Query has no functionality at all.
type NoStore struct{}

func (_ NoStore) Push(bp BundlePack) error {
	log.Printf("NoStore got pushed: %v", bp)
	return nil
}

func (_ NoStore) Query(_ func(BundlePack) bool) []BundlePack {
	return nil
}
