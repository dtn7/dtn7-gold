// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/dtn7/cboring"
)

// Sorted list of all known administrative record type codes to prevent double usage.
const (
	// AdminRecordTypeStatusReport is the administrative record type code for a status report.
	AdminRecordTypeStatusReport uint64 = 1
)

// AdministrativeRecord describes an administrative record, e.g., a status report.
type AdministrativeRecord interface {
	cboring.CborMarshaler

	// RecordTypeCode returns this AdministrativeRecord's type code.
	RecordTypeCode() uint64
}

// AdministrativeRecordManager keeps a book on various types of AdministrativeRecords that can be changed at runtime.
// Thus, new AdministrativeRecords can be created based on their block type code.
//
// A singleton AdministrativeRecordManager can be fetched by GetAdministrativeRecordManager.
type AdministrativeRecordManager struct {
	data sync.Map // map[record_type_code:uint64]reflect.Type
}

// NewAdministrativeRecordManager creates an empty AdministrativeRecordManager. To use a
// singleton AdministrativeRecordManager one can use GetAdministrativeRecordManager.
func NewAdministrativeRecordManager() *AdministrativeRecordManager {
	return &AdministrativeRecordManager{}
}

// Register a new AdministrativeRecord type through an exemplary instance.
func (arm *AdministrativeRecordManager) Register(ar AdministrativeRecord) (err error) {
	arCode := ar.RecordTypeCode()
	arType := reflect.TypeOf(ar).Elem()

	if otherAr, loaded := arm.data.LoadOrStore(arCode, arType); loaded {
		err = fmt.Errorf("record type code %d is already registered for %s",
			arCode, otherAr.(reflect.Type).Name())
	}

	return
}

// Unregister an AdministrativeRecord type through an exemplary instance.
func (arm *AdministrativeRecordManager) Unregister(ar AdministrativeRecord) {
	arm.data.Delete(ar.RecordTypeCode())
}

// IsKnown returns true if the AdministrativeRecord for this record type code is known.
func (arm *AdministrativeRecordManager) IsKnown(typeCode uint64) (known bool) {
	_, known = arm.data.Load(typeCode)
	return
}

// WriteAdministrativeRecord to a Writer wrapped in an CBOR array with both its record type code and its representation.
func (arm *AdministrativeRecordManager) WriteAdministrativeRecord(ar AdministrativeRecord, w io.Writer) (err error) {
	if err = cboring.WriteArrayLength(2, w); err != nil {
		return
	}

	if err = cboring.WriteUInt(ar.RecordTypeCode(), w); err != nil {
		return
	} else if cborErr := cboring.Marshal(ar, w); cborErr != nil {
		err = fmt.Errorf("marshalling AdministrativeRecord erred: %v", cborErr)
		return
	}

	return
}

// ReadAdministrativeRecord from a Reader within its CBOR array and returns the wrapped data type.
func (arm *AdministrativeRecordManager) ReadAdministrativeRecord(r io.Reader) (ar AdministrativeRecord, err error) {
	if n, cborErr := cboring.ReadArrayLength(r); cborErr != nil {
		err = cborErr
		return
	} else if n != 2 {
		err = fmt.Errorf("expected CBOR array of length 2, got %d", n)
		return
	}

	if typeCode, cborErr := cboring.ReadUInt(r); cborErr != nil {
		err = cborErr
		return
	} else if arType, ok := arm.data.Load(typeCode); !ok {
		err = fmt.Errorf("no AdministrativeRecord registered for record type code %d", typeCode)
		return
	} else {
		ar = reflect.New(arType.(reflect.Type)).Interface().(AdministrativeRecord)
		if cborErr := cboring.Unmarshal(ar, r); cborErr != nil {
			ar = nil
			err = fmt.Errorf("unmarshalling AdministrativeRecord with type code %d failed: %v", typeCode, cborErr)
			return
		}
	}

	return
}

var (
	administrativeRecordManager      *AdministrativeRecordManager
	administrativeRecordManagerMutex sync.Mutex
)

// GetAdministrativeRecordManager returns the singleton AdministrativeRecordManager.
// If none exists, a new one will be generated with "sane defaults".
func GetAdministrativeRecordManager() *AdministrativeRecordManager {
	administrativeRecordManagerMutex.Lock()
	defer administrativeRecordManagerMutex.Unlock()

	if administrativeRecordManager == nil {
		administrativeRecordManager = NewAdministrativeRecordManager()

		_ = administrativeRecordManager.Register(&StatusReport{})
	}

	return administrativeRecordManager
}

// NewAdministrativeRecordFromCbor creates a new AdministrativeRecord from a given byte array.
// TODO: remove this function; replace by AdministrativeRecordManager
func NewAdministrativeRecordFromCbor(data []byte) (ar AdministrativeRecord, err error) {
	buff := bytes.NewBuffer(data)
	return GetAdministrativeRecordManager().ReadAdministrativeRecord(buff)
}

// AdministrativeRecordToCbor creates a canonical block, containing this administrative record. The surrounding
// bundle _must_ have a set AdministrativeRecordPayload bundle processing control flag.
// TODO: remove this function; replace by AdministrativeRecordManager
func AdministrativeRecordToCbor(ar AdministrativeRecord) (blk CanonicalBlock, err error) {
	buff := new(bytes.Buffer)
	if err = GetAdministrativeRecordManager().WriteAdministrativeRecord(ar, buff); err != nil {
		return
	}

	blk = NewCanonicalBlock(1, 0, NewPayloadBlock(buff.Bytes()))
	return
}
