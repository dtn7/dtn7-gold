package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypeBundleAgeBlock uint64 = 8

type BundleAgeBlock uint64

func (bab *BundleAgeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeBundleAgeBlock
}

func NewBundleAgeBlock(us uint64) *BundleAgeBlock {
	bab := BundleAgeBlock(us)
	return &bab
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
