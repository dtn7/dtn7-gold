// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/storage"
)

// BundleDescriptor is a meta wrapper around a bpv7.Bundle to supply routing information without having to pass or
// alter the original bpv7.Bundle.
type BundleDescriptor struct {
	Id          bpv7.BundleID
	Receiver    bpv7.EndpointID
	Timestamp   time.Time
	Constraints map[Constraint]bool

	bndl  *bpv7.Bundle
	store *storage.Store
}

// NewBundleDescriptor for a bpv7.BundleID from a Store.
func NewBundleDescriptor(bid bpv7.BundleID, store *storage.Store) BundleDescriptor {
	descriptor := BundleDescriptor{
		Id:          bid,
		Receiver:    bpv7.DtnNone(),
		Timestamp:   time.Now(),
		Constraints: make(map[Constraint]bool),

		bndl:  nil,
		store: store,
	}

	if bi, err := descriptor.store.QueryId(descriptor.Id.Scrub()); err == nil {
		if v, ok := bi.Properties["bundlepack/receiver"]; ok {
			descriptor.Receiver = v.(bpv7.EndpointID)
		}
		if v, ok := bi.Properties["bundlepack/timestamp"]; ok {
			descriptor.Timestamp = v.(time.Time)
		}
		if v, ok := bi.Properties["bundlepack/constraints"]; ok {
			descriptor.Constraints = v.(map[Constraint]bool)
		}
	}

	return descriptor
}

// NewBundleDescriptorFromBundle for a bpv7.Bundle to be inserted into a Store.
func NewBundleDescriptorFromBundle(b bpv7.Bundle, store *storage.Store) BundleDescriptor {
	descriptor := NewBundleDescriptor(b.ID(), store)
	descriptor.bndl = &b

	_ = descriptor.Sync()
	return descriptor
}

// Sync this BundleDescriptor to the store.
func (descriptor BundleDescriptor) Sync() error {
	if !descriptor.store.KnowsBundle(descriptor.Id.Scrub()) {
		return descriptor.store.Push(*descriptor.bndl)
	} else if bi, err := descriptor.store.QueryId(descriptor.Id.Scrub()); err != nil {
		return err
	} else if len(descriptor.Constraints) == 0 {
		return descriptor.store.Delete(descriptor.Id)
	} else {
		bi.Pending = !descriptor.HasConstraint(ReassemblyPending) &&
			(descriptor.HasConstraint(ForwardPending) || descriptor.HasConstraint(Contraindicated))

		bi.Properties["bundlepack/receiver"] = descriptor.Receiver
		bi.Properties["bundlepack/timestamp"] = descriptor.Timestamp
		bi.Properties["bundlepack/constraints"] = descriptor.Constraints

		log.WithFields(log.Fields{
			"bundle":      descriptor.Id,
			"pending":     bi.Pending,
			"constraints": descriptor.Constraints,
		}).Debug("Synchronizing BundleDescriptor")

		updateErr := descriptor.store.Update(bi)
		if updateErr != nil {
			log.WithError(updateErr).Warn("Synchronizing errored")
		}
		return updateErr
	}
}

// Bundle returns this BundleDescriptor's internal bpv7.Bundle.
func (descriptor *BundleDescriptor) Bundle() (*bpv7.Bundle, error) {
	if descriptor.bndl != nil {
		return descriptor.bndl, nil
	}

	if bi, err := descriptor.store.QueryId(descriptor.Id.Scrub()); err != nil {
		return nil, err
	} else if bndl, err := bi.Parts[0].Load(); err != nil {
		return nil, err
	} else {
		descriptor.bndl = &bndl
		return &bndl, nil
	}
}

// MustBundle returns this BundleDescriptor's internal bpv7.Bundle or panics, compare the Bundle method.
func (descriptor *BundleDescriptor) MustBundle() *bpv7.Bundle {
	b, err := descriptor.Bundle()
	if err != nil {
		panic(err)
	}
	return b
}

// ID returns the wrapped Bundle's ID.
func (descriptor BundleDescriptor) ID() string {
	return descriptor.Id.String()
}

// HasReceiver returns true if this BundleDescriptor has a Receiver value.
func (descriptor BundleDescriptor) HasReceiver() bool {
	return !descriptor.Receiver.SameNode(bpv7.DtnNone())
}

// HasConstraint returns true if the given constraint contains.
func (descriptor BundleDescriptor) HasConstraint(c Constraint) bool {
	_, ok := descriptor.Constraints[c]
	return ok
}

// HasConstraints returns true if any constraint exists.
func (descriptor BundleDescriptor) HasConstraints() bool {
	return len(descriptor.Constraints) != 0
}

// AddConstraint adds the given constraint.
func (descriptor *BundleDescriptor) AddConstraint(c Constraint) {
	descriptor.Constraints[c] = true
}

// RemoveConstraint removes the given constraint.
func (descriptor *BundleDescriptor) RemoveConstraint(c Constraint) {
	delete(descriptor.Constraints, c)
}

// PurgeConstraints removes all constraints, except LocalEndpoint.
func (descriptor *BundleDescriptor) PurgeConstraints() {
	for c := range descriptor.Constraints {
		if c != LocalEndpoint {
			descriptor.RemoveConstraint(c)
		}
	}
}

// UpdateBundleAge updates the bundle's Bundle Age block based on its reception
// timestamp, if such a block exists.
func (descriptor *BundleDescriptor) UpdateBundleAge() (uint64, error) {
	bndl, err := descriptor.Bundle()
	if err != nil {
		return 0, err
	}

	ageBlock, err := bndl.ExtensionBlock(bpv7.ExtBlockTypeBundleAgeBlock)
	if err != nil {
		return 0, fmt.Errorf("no bundle age block exists")
	}

	age := ageBlock.Value.(*bpv7.BundleAgeBlock)
	return age.Increment(uint64(time.Since(descriptor.Timestamp)) / 1000), nil
}

func (descriptor BundleDescriptor) String() string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "BundleDescriptor(%v,", descriptor.ID())
	for c := range descriptor.Constraints {
		_, _ = fmt.Fprintf(&b, " %v", c)
	}
	if descriptor.HasReceiver() {
		_, _ = fmt.Fprintf(&b, ", %v", descriptor.Receiver)
	}
	_, _ = fmt.Fprintf(&b, ")")

	return b.String()
}
