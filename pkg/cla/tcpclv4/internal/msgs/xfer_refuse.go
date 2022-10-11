// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"encoding/binary"
	"fmt"
	"io"
)

// TransferRefusalCode is the one-octet refusal reason code for a XFER_REFUSE message.
type TransferRefusalCode uint8

const (
	// RefusalUnknown indicates an unknown or not specified reason.
	RefusalUnknown TransferRefusalCode = 0x00

	// RefusalCompleted indicates that the receiver already has the complete bundle.
	RefusalCompleted TransferRefusalCode = 0x01

	// RefusalNoResources indicate that the receiver's resources are exhausted.
	RefusalNoResources TransferRefusalCode = 0x02

	// RefusalRetransmit indicates a problem on the receiver's side.
	// This requires the complete bundle to be retransmitted.
	RefusalRetransmit TransferRefusalCode = 0x03

	// RefusalNotAcceptable indicates a problem regarding the bundle data or the transfer extension.
	// The sender should not retry the same transfer.
	RefusalNotAcceptable TransferRefusalCode = 0x04

	// RefusalExtensionFailure indicates a failure processing the Transfer Extension Items.
	RefusalExtensionFailure TransferRefusalCode = 0x05

	// RefusalSessionTerminating indicates the receiving entity is terminating this session.
	RefusalSessionTerminating TransferRefusalCode = 0x06
)

func (trc TransferRefusalCode) String() string {
	switch trc {
	case RefusalUnknown:
		return "Unknown"
	case RefusalCompleted:
		return "Completed"
	case RefusalNoResources:
		return "No Resources"
	case RefusalRetransmit:
		return "Retransmit"
	case RefusalNotAcceptable:
		return "Not Acceptable"
	case RefusalExtensionFailure:
		return "Extension Failure"
	case RefusalSessionTerminating:
		return "Session Terminating"
	default:
		return "INVALID"
	}
}

// IsValid checks if this TransferRefusalCode represents a valid value.
func (trc TransferRefusalCode) IsValid() bool {
	return trc.String() != "INVALID"
}

// XFER_REFUSE is the Message Header code for a Transfer Refusal Message.
const XFER_REFUSE uint8 = 0x03

// TransferRefusalMessage is the XFER_REFUSE message for transfer refusals.
type TransferRefusalMessage struct {
	ReasonCode TransferRefusalCode
	TransferId uint64
}

// NewTransferRefusalMessage creates a new TransferRefusalMessage with given fields.
func NewTransferRefusalMessage(reason TransferRefusalCode, tid uint64) *TransferRefusalMessage {
	return &TransferRefusalMessage{
		ReasonCode: reason,
		TransferId: tid,
	}
}

func (trm TransferRefusalMessage) String() string {
	return fmt.Sprintf("XFER_REFUSE(Reason Code=%v, Transfer iD=%d)", trm.ReasonCode, trm.TransferId)
}

func (trm TransferRefusalMessage) Marshal(w io.Writer) error {
	var fields = []interface{}{XFER_REFUSE, trm}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return err
		}
	}

	return nil
}

func (trm *TransferRefusalMessage) Unmarshal(r io.Reader) error {
	var messageHeader uint8
	if err := binary.Read(r, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != XFER_REFUSE {
		return fmt.Errorf("XFER_REFUSE's Message Header is wrong: %d instead of %d", messageHeader, XFER_REFUSE)
	}

	if err := binary.Read(r, binary.BigEndian, trm); err != nil {
		return err
	}

	if !trm.ReasonCode.IsValid() {
		return fmt.Errorf("XFER_REFUSE's Reason Code %x is invalid", trm.ReasonCode)
	}

	return nil
}
