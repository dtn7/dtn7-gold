// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"encoding/binary"
	"fmt"
	"io"
)

// SESS_INIT is the Message Header code for a Session Initialization Message.
const SESS_INIT uint8 = 0x07

// SessionInitMessage is the SESS_INIT message to negotiate session parameters.
//
// Session Extension Items are not yet implemented.
type SessionInitMessage struct {
	KeepaliveInterval uint16
	SegmentMru        uint64
	TransferMru       uint64
	NodeId            string

	// TODO: Session Extension Items
}

// NewSessionInitMessage creates a new SessionInitMessage with given fields.
func NewSessionInitMessage(keepaliveInterval uint16, segmentMru, transferMru uint64, nodeId string) *SessionInitMessage {
	return &SessionInitMessage{
		KeepaliveInterval: keepaliveInterval,
		SegmentMru:        segmentMru,
		TransferMru:       transferMru,
		NodeId:            nodeId,
	}
}

func (si SessionInitMessage) String() string {
	return fmt.Sprintf("SESS_INIT(Keepalive Interval=%d, Segment MRU=%d, Transfer MRU=%d, Node ID=%s)",
		si.KeepaliveInterval, si.SegmentMru, si.TransferMru, si.NodeId)
}

func (si SessionInitMessage) Marshal(w io.Writer) error {
	var fields = []interface{}{
		SESS_INIT,
		si.KeepaliveInterval,
		si.SegmentMru,
		si.TransferMru,
		uint16(len(si.NodeId))}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return err
		}
	}

	if n, err := io.WriteString(w, si.NodeId); err != nil {
		return err
	} else if n != len(si.NodeId) {
		return fmt.Errorf("SESS_INIT Node ID's length is %d, but only wrote %d bytes", len(si.NodeId), n)
	}

	// TODO: Session Extension Items
	// Currently, only an empty Session Extension Items Length is accepted.
	if err := binary.Write(w, binary.BigEndian, uint32(0)); err != nil {
		return err
	}

	return nil
}

func (si *SessionInitMessage) Unmarshal(r io.Reader) error {
	var messageHeader uint8
	if err := binary.Read(r, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != SESS_INIT {
		return fmt.Errorf("SESS_INIT's Message Header is wrong: %d instead of %d", messageHeader, SESS_INIT)
	}

	var nodeIdLen uint16
	var fields = []interface{}{
		&si.KeepaliveInterval,
		&si.SegmentMru,
		&si.TransferMru,
		&nodeIdLen,
	}

	for _, field := range fields {
		if err := binary.Read(r, binary.BigEndian, field); err != nil {
			return err
		}
	}

	var nodeIdBuff = make([]byte, nodeIdLen)
	if _, err := io.ReadFull(r, nodeIdBuff); err != nil {
		return err
	} else {
		si.NodeId = string(nodeIdBuff)
	}

	// TODO: Session Extension Items, see above
	var sessionExtsLen uint32
	if err := binary.Read(r, binary.BigEndian, &sessionExtsLen); err != nil {
		return err
	} else if sessionExtsLen > 0 {
		sessionExtsBuff := make([]byte, sessionExtsLen)

		if _, err := io.ReadFull(r, sessionExtsBuff); err != nil {
			return err
		}
	}

	return nil
}
