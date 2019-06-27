package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

const ExtBlockTypePayloadBlock uint64 = 1

type PayloadBlock []byte

func (pb *PayloadBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePayloadBlock
}

func NewPayloadBlock(data []byte) *PayloadBlock {
	pb := PayloadBlock(data)
	return &pb
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
