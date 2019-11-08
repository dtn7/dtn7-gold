package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypePreviousNodeBlock uint64 = 6

// PreviousNodeBlock implements the Bundle Protocol's Previous Node Block.
type PreviousNodeBlock EndpointID

func (pnb *PreviousNodeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePreviousNodeBlock
}

// NewPreviousNodeBlock creates a new Previous Node Block for an Endpoint ID.
func NewPreviousNodeBlock(prev EndpointID) *PreviousNodeBlock {
	pnb := PreviousNodeBlock(prev)
	return &pnb
}

// Endpoint returns this Previous Node Block's Endpoint ID.
func (pnb *PreviousNodeBlock) Endpoint() EndpointID {
	return EndpointID(*pnb)
}

func (pnb *PreviousNodeBlock) MarshalCbor(w io.Writer) error {
	endpoint := EndpointID(*pnb)
	return cboring.Marshal(&endpoint, w)
}

func (pnb *PreviousNodeBlock) UnmarshalCbor(r io.Reader) error {
	endpoint := EndpointID{}
	if err := cboring.Unmarshal(&endpoint, r); err != nil {
		return err
	} else {
		*pnb = PreviousNodeBlock(endpoint)
		return nil
	}
}

func (pb *PreviousNodeBlock) CheckValid() error {
	return EndpointID(*pb).CheckValid()
}
