package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypeBundleAgeBlock uint64 = 8

// BundleAgeBlock implements the Bundle Protocol's Bundle Age Block.
type BundleAgeBlock uint64

func (bab *BundleAgeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeBundleAgeBlock
}

// NewBundleAgeBlock creates a new BundleAgeBlock with the given microseconds.
func NewBundleAgeBlock(us uint64) *BundleAgeBlock {
	bab := BundleAgeBlock(us)
	return &bab
}

// Age returns the age in microseconds.
func (bab *BundleAgeBlock) Age() uint64 {
	return uint64(*bab)
}

// Increment with an offset in microseconds and return the new time.
func (bab *BundleAgeBlock) Increment(offset uint64) uint64 {
	newBabVal := uint64(*bab) + offset
	*bab = BundleAgeBlock(newBabVal)
	return newBabVal
}

func (bab *BundleAgeBlock) MarshalCbor(w io.Writer) error {
	return cboring.WriteUInt(uint64(*bab), w)
}

func (bab *BundleAgeBlock) UnmarshalCbor(r io.Reader) error {
	if us, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		*bab = BundleAgeBlock(us)
		return nil
	}
}

func (pb *BundleAgeBlock) CheckValid() error {
	return nil
}
