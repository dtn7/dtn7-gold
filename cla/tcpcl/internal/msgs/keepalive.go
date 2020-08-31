// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"encoding/binary"
	"fmt"
	"io"
)

// KEEPALIVE is the Message Header code for a Keepalive Message.
const KEEPALIVE uint8 = 0x04

// KeepaliveMessage is the KEEPALIVE message for session upkeep.
type KeepaliveMessage struct{}

// NewKeepaliveMessage creates a new KeepaliveMessage.
func NewKeepaliveMessage() *KeepaliveMessage {
	return &KeepaliveMessage{}
}

func (_ KeepaliveMessage) String() string {
	return "KEEPALIVE"
}

func (km KeepaliveMessage) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, KEEPALIVE)
}

func (km *KeepaliveMessage) Unmarshal(r io.Reader) error {
	var kmType uint8
	if err := binary.Read(r, binary.BigEndian, &kmType); err != nil {
		return err
	} else if kmType != KEEPALIVE {
		return fmt.Errorf("KEEPALIVE's value is %d instead of %d", kmType, KEEPALIVE)
	}

	return nil
}
