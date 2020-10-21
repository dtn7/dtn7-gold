// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"fmt"
	"io"
	"reflect"

	"github.com/dtn7/cboring"
)

// webAgentMessage describes a message which might be sent over a WebSocketAgent.
// Implementations are available at the end of this file.
type webAgentMessage interface {
	// typeCode is an unique identifier for each message type.
	// A const list of those and a map to a specific type will follow this interface's definition.
	typeCode() uint64

	// CborMarshaler must only be implemented for the type's logic.
	// A generic wrapper for the typeCode is available in the marshalCbor and unmarshalCbor functions.
	cboring.CborMarshaler
}

const (
	wamStatusCode          uint64 = 0
	wamRegisterCode        uint64 = 1
	wamBundleCode          uint64 = 2
	wamSyscallRequestCode  uint64 = 3
	wamSyscallResponseCode uint64 = 4
)

var wamMapping = map[interface{}]reflect.Type{
	wamStatusCode:          reflect.TypeOf(wamStatus{}),
	wamRegisterCode:        reflect.TypeOf(wamRegister{}),
	wamBundleCode:          reflect.TypeOf(wamBundle{}),
	wamSyscallRequestCode:  reflect.TypeOf(wamSyscallRequest{}),
	wamSyscallResponseCode: reflect.TypeOf(wamSyscallResponse{}),
}

// marshalCbor writes a webAgentMessage wrapped with its type code as CBOR.
func marshalCbor(wam webAgentMessage, w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	if err := cboring.WriteUInt(wam.typeCode(), w); err != nil {
		return err
	}

	if err := cboring.Marshal(wam, w); err != nil {
		return err
	}

	return nil
}

// unmarshalCbor reads a new webAgentMessage based on its type code from CBOR.
func unmarshalCbor(r io.Reader) (wam webAgentMessage, err error) {
	if n, arrErr := cboring.ReadArrayLength(r); arrErr != nil {
		err = arrErr
		return
	} else if n != 2 {
		err = fmt.Errorf("expected array of two elements, got %d", n)
		return
	}

	if n, typeErr := cboring.ReadUInt(r); typeErr != nil {
		err = typeErr
		return
	} else if t, ok := wamMapping[n]; !ok {
		err = fmt.Errorf("no known WAM type code %d", n)
		return
	} else {
		wam = reflect.New(t).Interface().(webAgentMessage)
	}

	if wamErr := cboring.Unmarshal(wam, r); wamErr != nil {
		err = wamErr
		return
	}

	return
}
