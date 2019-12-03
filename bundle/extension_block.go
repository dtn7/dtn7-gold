package bundle

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/dtn7/cboring"
)

// ExtensionBlock is a specific shape of a Canonical Block, i.e., the Payload
// Block or a more generic Extension Block as defined in section 4.3.
type ExtensionBlock interface {
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
	data  map[uint64]reflect.Type
	mutex sync.Mutex
}

// NewExtensionBlockManager creates an empty ExtensionBlockManager. To use a
// singleton ExtensionBlockManager one can use GetExtensionBlockManager.
func NewExtensionBlockManager() *ExtensionBlockManager {
	return &ExtensionBlockManager{
		data: make(map[uint64]reflect.Type),
	}
}

// Register a new ExtensionBlock type through an exemplary instance.
func (ebm *ExtensionBlockManager) Register(eb ExtensionBlock) error {
	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	extCode := eb.BlockTypeCode()
	extType := reflect.TypeOf(eb).Elem()

	if extType == reflect.TypeOf((*GenericExtensionBlock)(nil)).Elem() {
		return fmt.Errorf("not allowed to register a GenericExtensionBlock")
	}

	if otherType, exists := ebm.data[extCode]; exists {
		return fmt.Errorf("Block type code %d is already registered for %s",
			extCode, otherType.Name())
	}

	ebm.data[extCode] = extType
	return nil
}

// Register an ExtensionBlock type through an exemplary instance.
func (ebm *ExtensionBlockManager) Unregister(eb ExtensionBlock) {
	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	delete(ebm.data, eb.BlockTypeCode())
}

// IsKnown returns true if the ExtensionBlock for this block type code is known.
func (ebm *ExtensionBlockManager) IsKnown(typeCode uint64) bool {
	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	_, known := ebm.data[typeCode]
	return known
}

// createBlock returns either a specific ExtensionBlock or, if type code is not registered, an GenericExtensionBlock.
func (ebm *ExtensionBlockManager) createBlock(typeCode uint64) ExtensionBlock {
	if extType, exists := ebm.data[typeCode]; exists {
		return reflect.New(extType).Interface().(ExtensionBlock)
	} else {
		return &GenericExtensionBlock{typeCode: typeCode}
	}
}

// WriteBlock writes an ExtensionBlock in its correct binary format into the io.Writer.
// Unknown block types are treated as GenericExtensionBlock.
func (ebm *ExtensionBlockManager) WriteBlock(b ExtensionBlock, w io.Writer) error {
	switch b.(type) {
	// TODO: binary encoding case

	case cboring.CborMarshaler:
		// If the Block can be encoded as CBOR, then this encoding is wrapped in another CBOR byte string.
		var buff bytes.Buffer
		if err := cboring.Marshal(b.(cboring.CborMarshaler), &buff); err != nil {
			return fmt.Errorf("marshalling CBOR for Block errored: %v", err)
		}
		return cboring.WriteByteString(buff.Bytes(), w)

	default:
		return fmt.Errorf("ExtensionBlock does not implement any expected types")
	}
}

// ReadBlock reads an ExtensionBlock from its correct binary format from the io.Reader.
// Unknown block types are treated as GenericExtensionBlock.
func (ebm *ExtensionBlockManager) ReadBlock(typeCode uint64, r io.Reader) (b ExtensionBlock, err error) {
	b = ebm.createBlock(typeCode)

	switch b.(type) {
	// TODO: binary decoding case

	case cboring.CborMarshaler:
		// A CBOR encoded block is wrapped in a CBOR byte string itself.
		if data, dataErr := cboring.ReadByteString(r); dataErr != nil {
			err = dataErr
		} else {
			var buff = bytes.NewBuffer(data)
			err = cboring.Unmarshal(b.(cboring.CborMarshaler), buff)
		}

	default:
		err = fmt.Errorf("ExtensionBlock does not implement any expected types")
	}

	return
}

var (
	extensionBlockManager      *ExtensionBlockManager
	extensionBlockManagerMutex sync.Mutex
)

// GetExtensionBlockManager returns the singleton ExtensionBlockManager. If none
// exists, a new ExtensionBlockManager will be generated with a knowledge of the
// PayloadBlock, PreviousNodeBlock, BundleAgeBlock and HopCountBlock.
func GetExtensionBlockManager() *ExtensionBlockManager {
	extensionBlockManagerMutex.Lock()
	defer extensionBlockManagerMutex.Unlock()

	if extensionBlockManager == nil {
		extensionBlockManager = NewExtensionBlockManager()

		_ = extensionBlockManager.Register(NewPayloadBlock(nil))
		_ = extensionBlockManager.Register(NewPreviousNodeBlock(DtnNone()))
		_ = extensionBlockManager.Register(NewBundleAgeBlock(0))
		_ = extensionBlockManager.Register(NewHopCountBlock(0))
	}

	return extensionBlockManager
}
