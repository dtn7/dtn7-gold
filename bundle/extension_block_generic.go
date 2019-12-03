package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

// GenericExtensionBlock is a dummy ExtensionBlock to cover for unknown or unregistered ExtensionBlocks.
type GenericExtensionBlock struct {
	data     []byte
	typeCode uint64
}

// NewGenericExtensionBlock creates a new GenericExtensionBlock from some payload and a block type code.
func NewGenericExtensionBlock(data []byte, typeCode uint64) *GenericExtensionBlock {
	return &GenericExtensionBlock{
		data:     data,
		typeCode: typeCode,
	}
}

func (geb *GenericExtensionBlock) MarshalCbor(w io.Writer) error {
	return cboring.WriteByteString(geb.data, w)
}

func (geb *GenericExtensionBlock) UnmarshalCbor(r io.Reader) (err error) {
	geb.data, err = cboring.ReadByteString(r)
	return
}

func (geb *GenericExtensionBlock) CheckValid() error {
	// We have zero knowledge about this block.
	// Thus, who are we to judge someone else's block?
	return nil
}

func (geb *GenericExtensionBlock) BlockTypeCode() uint64 {
	return geb.typeCode
}
