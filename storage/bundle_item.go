package storage

import (
	"crypto/sha1"
	"fmt"
	"path"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
)

type BundleItem struct {
	Id string `badgerhold:"key"`

	Pending bool      `badgerholdIndex:"Pending"`
	Expires time.Time `badgerholdIndex:"Expires"`

	Fragmented bool
	Parts      []BundlePart

	Properties map[string]interface{}
}

type BundlePart struct {
	Filename string

	FragmentOffset  uint64
	TotalDataLength uint64
}

func calcExpirationDate(b bundle.Bundle) time.Time {
	// TODO: check Bundle Age Block
	return b.PrimaryBlock.CreationTimestamp.DtnTime().Time().Add(
		time.Duration(b.PrimaryBlock.Lifetime) * time.Microsecond)
}

func bundlePartPath(id bundle.BundleID, storagePath string) string {
	f := fmt.Sprintf("%x", sha1.Sum([]byte(id.String())))
	return path.Join(storagePath, f)
}

func NewBundleItem(b bundle.Bundle, storagePath string) (bi BundleItem) {
	bid := b.ID()

	bi = BundleItem{
		Id: bid.Scrub().String(),

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
