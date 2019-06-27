package bundle

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypeHopCountBlock uint64 = 9

type HopCountBlock struct {
	Limit uint64
	Count uint64
}

func (hcb *HopCountBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeHopCountBlock
}

func NewHopCountBlock(limit uint64) *HopCountBlock {
	hcb := HopCountBlock{limit, 0}
	return &hcb
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

	fields := []uint64{hcb.Limit, hcb.Count}
	for _, f := range fields {
		if err := cboring.WriteUInt(f, w); err != nil {
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

	fields := []*uint64{&hcb.Limit, &hcb.Count}
	for _, f := range fields {
		if x, err := cboring.ReadUInt(r); err != nil {
			return err
		} else {
			*f = x
		}
	}

	return nil
}
