package tcpcl

import (
	"encoding/binary"
	"fmt"
	"io"
)

// KEEPALIVE is the Message Header code for a Keepalive Message.
const KEEPALIVE uint8 = 0x04

// KeepaliveMessage is the KEEPALIVE message for session upkeep.
type KeepaliveMessage uint8

// NewKeepaliveMessage creates a new KeepaliveMessage.
func NewKeepaliveMessage() KeepaliveMessage {
	return KeepaliveMessage(KEEPALIVE)
}

func (_ KeepaliveMessage) String() string {
	return "KEEPALIVE"
}

func (km KeepaliveMessage) Marshal(w io.Writer) error {
	if uint8(km) != KEEPALIVE {
		return fmt.Errorf("KEEPALIVE's value is %d instead of %d", uint8(km), KEEPALIVE)
	}

	return binary.Write(w, binary.BigEndian, km)
}

func (km *KeepaliveMessage) Unmarshal(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, km); err != nil {
		return err
	}

	if uint8(*km) != KEEPALIVE {
		return fmt.Errorf("KEEPALIVE's value is %d instead of %d", uint8(*km), KEEPALIVE)
	}

	return nil
}
