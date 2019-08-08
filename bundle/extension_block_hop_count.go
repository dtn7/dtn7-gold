package bundle

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypeHopCountBlock uint64 = 9

// HopCountBlock implements the Bundle Protocol's Hop Count Block.
type HopCountBlock struct {
	Limit uint8
	Count uint8
}

func (hcb *HopCountBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeHopCountBlock
}

// NewHopCountBlock creates a new HopCountBlock with a given hop limit.
func NewHopCountBlock(limit uint8) *HopCountBlock {
	return &HopCountBlock{
		Limit: limit,
		Count: 0,
	}
}

// IsExceeded returns true if the hop limit exceeded.
func (hcb HopCountBlock) IsExceeded() bool {
	return hcb.Count > hcb.Limit
}

// Increment the hop counter and returns if the hop limit is exceeded afterwards.
func (hcb *HopCountBlock) Increment() bool {
	hcb.Count++

	return hcb.IsExceeded()
}

// Decrement the hop counter.
func (hcb *HopCountBlock) Decrement() {
	hcb.Count--
}

func (hcb *HopCountBlock) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	fields := []uint8{hcb.Limit, hcb.Count}
	for _, f := range fields {
		if err := cboring.WriteUInt(uint64(f), w); err != nil {
			return err
		}
	}

	return nil
}

func (hcb *HopCountBlock) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("Expected array with length 2, got %d", l)
	}

	fields := []*uint8{&hcb.Limit, &hcb.Count}
	for _, f := range fields {
		if x, err := cboring.ReadUInt(r); err != nil {
			return err
		} else if x > 255 {
			return fmt.Errorf("Hop Count fields must be within a range to 255, not %d", x)
		} else {
			*f = uint8(x)
		}
	}

	return nil
}

func (pb *HopCountBlock) CheckValid() error {
	if pb.IsExceeded() {
		return fmt.Errorf("HopCountBlock is exceeded")
	}
	return nil
}
