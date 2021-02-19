// SPDX-FileCopyrightText: 2019, 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// BundleItem is a wrapper for meta data around a Bundle. The Store operates
// on BundleItems instead of Bundles.
type BundleItem struct {
	Id  string `badgerhold:"key"`
	BId bpv7.BundleID

	Pending bool      `badgerholdIndex:"Pending"`
	Expires time.Time `badgerholdIndex:"Expires"`

	Fragmented bool
	Parts      []BundlePart

	Properties map[string]interface{}
}

// bundleParts is a slice of loaded bundleParts.
func (bi BundleItem) bundleParts() (bundleParts []bpv7.Bundle, err error) {
	bundleParts = make([]bpv7.Bundle, len(bi.Parts))
	for i, part := range bi.Parts {
		if bundleParts[i], err = part.Load(); err != nil {
			return
		}
	}
	return
}

// Load the complete bpv7.Bundle for a BundleItem. If there are multiple fragments, a reassembly will be performed.
func (bi BundleItem) Load() (b bpv7.Bundle, err error) {
	var parts []bpv7.Bundle
	if parts, err = bi.bundleParts(); err == nil {
		b, err = bpv7.ReassembleFragments(parts)
	}
	return
}

// IsComplete determines if the BundleItem is complete and can be Load()ed.
func (bi BundleItem) IsComplete() bool {
	if !bi.Fragmented {
		return true
	}

	parts, err := bi.bundleParts()
	return err == nil && bpv7.IsBundleReassemblable(parts)
}

// BundlePart links a BundleItem to a Bundle with possible information
// regarding fragmentation.
type BundlePart struct {
	Filename string

	FragmentOffset  uint64
	TotalDataLength uint64
}

// storeBundle serializes the Bundle of a BundleItem/BundlePart to the disk.
func (bp BundlePart) storeBundle(b bpv7.Bundle) error {
	if f, err := os.OpenFile(bp.Filename, os.O_WRONLY|os.O_CREATE, 0600); err != nil {
		return err
	} else {
		return b.WriteBundle(f)
	}
}

// deleteBundle removes the serialized Bundle from the disk.
func (bp BundlePart) deleteBundle() error {
	return os.Remove(bp.Filename)
}

// Load the Bundle struct from the disk.
func (bp BundlePart) Load() (b bpv7.Bundle, err error) {
	if f, fErr := os.Open(bp.Filename); fErr != nil {
		err = fErr
	} else {
		b, err = bpv7.ParseBundle(f)
	}
	return
}

// calcExpirationDate for a Bundle.
func calcExpirationDate(b bpv7.Bundle) time.Time {
	// TODO: check Bundle Age Block
	return b.PrimaryBlock.CreationTimestamp.DtnTime().Time().Add(
		time.Duration(b.PrimaryBlock.Lifetime) * time.Millisecond)
}

// bundlePartPath returns a path for a Bundle.
func bundlePartPath(id bpv7.BundleID, storagePath string) string {
	f := fmt.Sprintf("%x", sha256.Sum256([]byte(id.String())))
	return path.Join(storagePath, f)
}

// newBundleItem creates a new BundleItem for a Bundle.
func newBundleItem(b bpv7.Bundle, storagePath string) (bi BundleItem) {
	bid := b.ID()

	bi = BundleItem{
		Id:  bid.Scrub().String(),
		BId: bid.Scrub(),

		Pending: false,
		Expires: calcExpirationDate(b),

		Fragmented: b.PrimaryBlock.HasFragmentation(),

		Properties: make(map[string]interface{}),
	}

	bp := BundlePart{
		Filename: bundlePartPath(bid, storagePath),

		FragmentOffset:  bid.FragmentOffset,
		TotalDataLength: bid.TotalDataLength,
	}

	bi.Parts = append(bi.Parts, bp)

	return
}
