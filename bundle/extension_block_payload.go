package bundle

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
	return *pb
}

func (pb *PayloadBlock) MarshalBinary() ([]byte, error) {
	return *pb, nil
}

func (pb *PayloadBlock) UnmarshalBinary(data []byte) error {
	*pb = data
	return nil
}

func (pb *PayloadBlock) CheckValid() error {
	return nil
}
