// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/bundle"
)

// TransferMessage transmits a Bundle to the peer.
//
// To verify a transfer, an identifier is used. After successfully reception, a TransferAckMessage will be sent back
// referring this identifier.
type TransferMessage struct {
	Identifier uint64
	Bundle     bundle.Bundle
}

// NewTransferMessage for a Bundle to be sent. An identifier will be generated.
func NewTransferMessage(b bundle.Bundle) (tm *TransferMessage, err error) {
	idBytes := make([]byte, 8)
	if _, err = rand.Read(idBytes); err != nil {
		return
	}

	id, idErr := binary.ReadUvarint(bytes.NewBuffer(idBytes))
	if idErr != nil {
		err = idErr
		return
	}

	tm = &TransferMessage{
		Identifier: id,
		Bundle:     b,
	}
	return
}

// Type code of a TransferMessage is always 2.
func (tm *TransferMessage) Type() uint64 {
	return MsgTransfer
}

func (tm *TransferMessage) String() string {
	return fmt.Sprintf("TransferMessage(%d,%s)", tm.Identifier, tm.Bundle.String())
}

// MarshalCbor creates a CBOR array of two elements: the uint identifier and the Bundle's CBOR representation.
func (tm *TransferMessage) MarshalCbor(w io.Writer) (err error) {
	if err = cboring.WriteArrayLength(2, w); err != nil {
		return
	}

	if err = cboring.WriteUInt(tm.Identifier, w); err != nil {
		return
	}
	if err = cboring.Marshal(&tm.Bundle, w); err != nil {
		return
	}

	return
}

// UnmarshalCbor a CBOR array back to a TransferMessage.
func (tm *TransferMessage) UnmarshalCbor(r io.Reader) (err error) {
	if n, arrErr := cboring.ReadArrayLength(r); arrErr != nil {
		return arrErr
	} else if n != 2 {
		return fmt.Errorf("TransferMessage expected array length of 2, got %d elements", n)
	}

	if tm.Identifier, err = cboring.ReadUInt(r); err != nil {
		return
	}
	if err = cboring.Unmarshal(&tm.Bundle, r); err != nil {
		return
	}

	return
}
