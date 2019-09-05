package tcpcl

import "fmt"

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

// MarshalBinary encodes this KeepaliveMessage into its binary form.
func (km KeepaliveMessage) MarshalBinary() (data []byte, err error) {
	if uint8(km) != KEEPALIVE {
		err = fmt.Errorf("KEEPALIVE's value is %d instead of %d", uint8(km), KEEPALIVE)
		return
	}

	data = []byte{KEEPALIVE}
	return
}

// UnmarshalBinary decodes a KeepaliveMessage from its binary form.
func (km *KeepaliveMessage) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return fmt.Errorf("KEEPALIVE's octet length is %d instead of 1", len(data))
	}

	if x := uint8(data[0]); x != KEEPALIVE {
		return fmt.Errorf("KEEPALIVE's value is %d instead of %d", x, KEEPALIVE)
	} else {
		*km = KeepaliveMessage(x)
	}

	return nil
}
