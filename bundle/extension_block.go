// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/dtn7/cboring"
)

// Sorted list of all known block type codes to prevent double usage.
const (
	// ExtBlockTypePayloadBlock is the block type code for a Payload Block, bundle/extension_block_payload.go
	ExtBlockTypePayloadBlock uint64 = 1

	// ExtBlockTypePreviousNodeBlock is the block type code for a Previous Node Block, bundle/extension_block_previous_node.go
	ExtBlockTypePreviousNodeBlock uint64 = 6

	// ExtBlockTypeBundleAgeBlock is the block type code for a Bundle Age Block, bundle/extension_block_bundle_age.go
	ExtBlockTypeBundleAgeBlock uint64 = 7

	// ExtBlockTypeHopCountBlock is the block type code for a Hop Count Block, bundle/extension_block_hop_count.go
	ExtBlockTypeHopCountBlock uint64 = 10

	// ExtBlockTypeBinarySprayBlock is the custom block type code for a BinarySprayBlock, core/routing_spray.go
	ExtBlockTypeBinarySprayBlock uint64 = 192

	// ExtBlockTypeDTLSRBlock is the custom block type code for a DTLSRBlock, core/routing_dtlsr.go
	ExtBlockTypeDTLSRBlock uint64 = 193

	// ExtBlockTypeProphetBlock is the custom block type code for a ProphetBlock, core/routing_prophet.go
	ExtBlockTypeProphetBlock uint64 = 194

	// ExtBlockTypeSignatureBlock is the custom block type code for a SignatureBlock, bundle/extension_block_signature.go
	ExtBlockTypeSignatureBlock uint64 = 195
)

// ExtensionBlock describes the block-type specific data of any Canonical Block. Such an ExtensionBlock
// must implement either the cboring.CborMarshaler interface, if its serializable to / from CBOR, or both
// encoding.BinaryMarshaler and encoding.BinaryUnmarshaler. The latter allows any kind of serialization,
// e.g., to a totally custom format.
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
		return fmt.Errorf("block type code %d is already registered for %s",
			extCode, otherType.Name())
	}

	ebm.data[extCode] = extType
	return nil
}

// Unregister an ExtensionBlock type through an exemplary instance.
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
	switch b := b.(type) {
	case encoding.BinaryMarshaler:
		if data, err := b.MarshalBinary(); err != nil {
			return fmt.Errorf("marshalling binary for Block errored: %v", err)
		} else {
			return cboring.WriteByteString(data, w)
		}

	case cboring.CborMarshaler:
		var buff bytes.Buffer
		if err := cboring.Marshal(b, &buff); err != nil {
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

	switch b := b.(type) {
	case encoding.BinaryUnmarshaler:
		if data, dataErr := cboring.ReadByteString(r); dataErr != nil {
			err = dataErr
		} else {
			err = b.UnmarshalBinary(data)
		}

	case cboring.CborMarshaler:
		if data, dataErr := cboring.ReadByteString(r); dataErr != nil {
			err = dataErr
		} else {
			var buff = bytes.NewBuffer(data)
			err = cboring.Unmarshal(b, buff)
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
