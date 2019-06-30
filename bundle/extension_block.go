package bundle

import (
	"fmt"
	"reflect"

	"github.com/dtn7/cboring"
)

// ExtensionBlock is a specific shape of a Canonical Block, i.e., the Payload
// Block or a more generic Extension Block as defined in section 4.3.
type ExtensionBlock interface {
	cboring.CborMarshaler
	Valid

	// BlockTypeCode must return a constant integer, indicating the block type code.
	BlockTypeCode() uint64
}

// ExtensionBlockManager keeps a book on various types of ExtensionBlocks that
// can be changed at runtime. Thus, new ExtensionBlocks can be created based on
// their block type code.
//
// A singleton ExtensionBlockManager can be fetched by GetExtensionBlockManager.
type ExtensionBlockManager struct {
	data map[uint64]reflect.Type
}

// NewExtensionBlockManager creates an empty ExtensionBlockManager. To use a
// singleton ExtensionBlockManager one can use GetExtensionBlockManager.
func NewExtensionBlockManager() *ExtensionBlockManager {
	return &ExtensionBlockManager{make(map[uint64]reflect.Type)}
}

// Register a new ExtensionBlock type through an exemplary instance.
func (ebm *ExtensionBlockManager) Register(eb ExtensionBlock) error {
	extCode := eb.BlockTypeCode()
	extType := reflect.TypeOf(eb).Elem()

	if otherType, exists := ebm.data[extCode]; exists {
		return fmt.Errorf("Block type code %d is already registered for %s",
			extCode, otherType.Name())
	}

	ebm.data[extCode] = extType
	return nil
}

// Register an ExtensionBlock type through an exemplary instance.
func (ebm *ExtensionBlockManager) Unregister(eb ExtensionBlock) {
	delete(ebm.data, eb.BlockTypeCode())
}

// CreateBlock returns an instance of the ExtensionBlock for the requested
// block type code.
func (ebm *ExtensionBlockManager) CreateBlock(typeCode uint64) (eb ExtensionBlock, err error) {
	extType, exists := ebm.data[typeCode]
	if !exists {
		err = fmt.Errorf("No ExtensionBlock for block type code %d", typeCode)
		return
	}

	eb = reflect.New(extType).Interface().(ExtensionBlock)
	return
}

// extensionBlockManager is the pointer to the singleton ExtensionBlockManager
var extensionBlockManager *ExtensionBlockManager

// GetExtensionBlockManager returns the singleton ExtensionBlockManager. If none
// exists, a new ExtensionBlockManager will be generated with a knowledge of the
// PayloadBlock, PreviousNodeBlock, BundleAgeBlock and HopCountBlock.
func GetExtensionBlockManager() *ExtensionBlockManager {
	if extensionBlockManager == nil {
		extensionBlockManager = NewExtensionBlockManager()

		extensionBlockManager.Register(NewPayloadBlock(nil))
		extensionBlockManager.Register(NewPreviousNodeBlock(DtnNone()))
		extensionBlockManager.Register(NewBundleAgeBlock(0))
		extensionBlockManager.Register(NewHopCountBlock(0))
	}

	return extensionBlockManager
}
