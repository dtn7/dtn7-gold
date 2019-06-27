package bundle

import "github.com/dtn7/cboring"

// ExtensionBlock is a specific shape of a Canonical Block, i.e., the Payload
// Block or a more generic Extension Block as defined in section 4.3.
type ExtensionBlock interface {
	cboring.CborMarshaler

	// BlockTypeCode must return a constant integer, indicating the block type code.
	BlockTypeCode() uint64
}
