package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypePayloadBlock uint64 = 1

// PayloadBlock implements the Bundle Protocol's Payload Block.
type PayloadBlock []byte

func (pb *PayloadBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePayloadBlock
}

// NewPayloadBlock creates a new PayloadBlock with the given payload.
func NewPayloadBlock(data []byte) *PayloadBlock {
	pb := PayloadBlock(data)
	return &pb
}

// Data returns this PayloadBlock's payload.
func (pb *PayloadBlock) Data() []byte {
	return []byte(*pb)
}

func (pb *PayloadBlock) MarshalCbor(w io.Writer) error {
	return cboring.WriteByteString([]byte(*pb), w)
}

func (pb *PayloadBlock) UnmarshalCbor(r io.Reader) error {
	if pl, err := cboring.ReadByteString(r); err != nil {
		return err
	} else {
		*pb = PayloadBlock(pl)
		return nil
	}
}

func (pb *PayloadBlock) CheckValid() error {
	return nil
}
