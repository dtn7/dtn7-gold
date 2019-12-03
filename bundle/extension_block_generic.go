package bundle

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

func (geb *GenericExtensionBlock) MarshalBinary() ([]byte, error) {
	return geb.data, nil
}

func (geb *GenericExtensionBlock) UnmarshalBinary(data []byte) error {
	geb.data = data
	return nil
}

func (geb *GenericExtensionBlock) CheckValid() error {
	// We have zero knowledge about this block.
	// Thus, who are we to judge someone else's block?
	return nil
}

func (geb *GenericExtensionBlock) BlockTypeCode() uint64 {
	return geb.typeCode
}
