// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/storage"
)

// BundlePack is a set of a bundle, it's creation or reception time stamp and
// a set of constraints used in the process of delivering this bundle.
type BundlePack struct {
	Id          bundle.BundleID
	Receiver    bundle.EndpointID
	Timestamp   time.Time
	Constraints map[Constraint]bool

	bndl  *bundle.Bundle
	store *storage.Store
}

// NewBundlePack returns a BundlePack for the given bundle.
func NewBundlePack(bid bundle.BundleID, store *storage.Store) BundlePack {
	bp := BundlePack{
		Id:          bid,
		Receiver:    bundle.DtnNone(),
		Timestamp:   time.Now(),
		Constraints: make(map[Constraint]bool),

		bndl:  nil,
		store: store,
	}

	if bi, err := bp.store.QueryId(bp.Id.Scrub()); err == nil {
		if v, ok := bi.Properties["bundlepack/receiver"]; ok {
			bp.Receiver = v.(bundle.EndpointID)
		}
		if v, ok := bi.Properties["bundlepack/timestamp"]; ok {
			bp.Timestamp = v.(time.Time)
		}
		if v, ok := bi.Properties["bundlepack/constraints"]; ok {
			bp.Constraints = v.(map[Constraint]bool)
		}
	}

	return bp
}

func NewBundlePackFromBundle(b bundle.Bundle, store *storage.Store) BundlePack {
	bp := NewBundlePack(b.ID(), store)

	bp.bndl = &b

	_ = bp.Sync()
	return bp
}

// Sync this BundlePack to the store.
func (bp BundlePack) Sync() error {
	if !bp.store.KnowsBundle(bp.Id.Scrub()) {
		return bp.store.Push(*bp.bndl)
	} else if bi, err := bp.store.QueryId(bp.Id.Scrub()); err != nil {
		return err
	} else if len(bp.Constraints) == 0 {
		return bp.store.Delete(bp.Id)
	} else {
		bi.Pending = !bp.HasConstraint(ReassemblyPending) &&
			(bp.HasConstraint(ForwardPending) || bp.HasConstraint(Contraindicated))

		bi.Properties["bundlepack/receiver"] = bp.Receiver
		bi.Properties["bundlepack/timestamp"] = bp.Timestamp
		bi.Properties["bundlepack/constraints"] = bp.Constraints

		log.WithFields(log.Fields{
			"bundle":      bp.Id,
			"pending":     bi.Pending,
			"constraints": bp.Constraints,
		}).Debug("Sync'ing BundlePack")

		updateErr := bp.store.Update(bi)
		if updateErr != nil {
			log.WithFields(log.Fields{
				"error": updateErr,
			}).Info("Sync'ing failed")
		}
		return updateErr
	}
}

// Bundle returns this BundlePack's Bundle.
func (bp *BundlePack) Bundle() (*bundle.Bundle, error) {
	if bp.bndl != nil {
		return bp.bndl, nil
	}

	if bi, err := bp.store.QueryId(bp.Id.Scrub()); err != nil {
		return nil, err
	} else if bndl, err := bi.Parts[0].Load(); err != nil {
		return nil, err
	} else {
		bp.bndl = &bndl
		return &bndl, nil
	}
}

// MustBundle: like Bundle, just more stupid
func (bp *BundlePack) MustBundle() *bundle.Bundle {
	b, err := bp.Bundle()
	if err != nil {
		panic(err)
	}
	return b
}

// ID returns the wrapped Bundle's ID.
func (bp BundlePack) ID() string {
	return bp.Id.String()
}

// HasReceiver returns true if this BundlePack has a Receiver value.
func (bp BundlePack) HasReceiver() bool {
	return bp.Receiver != bundle.DtnNone()
}

// HasConstraint returns true if the given constraint contains.
func (bp BundlePack) HasConstraint(c Constraint) bool {
	_, ok := bp.Constraints[c]
	return ok
}

// HasConstraints returns true if any constraint exists.
func (bp BundlePack) HasConstraints() bool {
	return len(bp.Constraints) != 0
}

// AddConstraint adds the given constraint.
func (bp *BundlePack) AddConstraint(c Constraint) {
	bp.Constraints[c] = true
}

// RemoveConstraint removes the given constraint.
func (bp *BundlePack) RemoveConstraint(c Constraint) {
	delete(bp.Constraints, c)
}

// PurgeConstraints removes all constraints, except LocalEndpoint.
func (bp *BundlePack) PurgeConstraints() {
	for c := range bp.Constraints {
		if c != LocalEndpoint {
			bp.RemoveConstraint(c)
		}
	}
}

// UpdateBundleAge updates the bundle's Bundle Age block based on its reception
// timestamp, if such a block exists.
func (bp *BundlePack) UpdateBundleAge() (uint64, error) {
	bndl, err := bp.Bundle()
	if err != nil {
		return 0, err
	}

	ageBlock, err := bndl.ExtensionBlock(bundle.ExtBlockTypeBundleAgeBlock)
	if err != nil {
		return 0, newCoreError("No such block")
	}

	age := ageBlock.Value.(*bundle.BundleAgeBlock)
	return age.Increment(uint64(time.Since(bp.Timestamp)) / 1000), nil
}

func (bp BundlePack) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "BundlePack(%v,", bp.ID())

	for c := range bp.Constraints {
		fmt.Fprintf(&b, " %v", c)
	}

	if bp.HasReceiver() {
		fmt.Fprintf(&b, ", %v", bp.Receiver)
	}

	fmt.Fprintf(&b, ")")

	return b.String()
}
