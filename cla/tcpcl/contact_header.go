package tcpcl

import (
	"bytes"
	"fmt"
)

// ContactFlags are single-bit flags used in the ContactHeader.
type ContactFlags uint8

const (
	// ContactFlags_NONE is a default for no flags
	ContactFlags_NONE ContactFlags = 0x00

	// ContactFlags_CAN_TLS indicates that the sending peer is capable of TLS security.
	ContactFlags_CAN_TLS ContactFlags = 0x01

	// contactFlags_INVALID is a bit field of all invalid ContactFlags.
	contactFlags_INVALID ContactFlags = 0xFE
)

func (cf ContactFlags) String() string {
	switch cf {
	case ContactFlags_NONE:
		return "NONE"
	case ContactFlags_CAN_TLS:
		return "CAN_TLS"
	default:
		return "INVALID"
	}
}

// ContactHeader will be exchanged at first after a TCP connection was
// established. Both entities are sending a ContactHeader and are validating
// the peer's one.
type ContactHeader struct {
	Flags ContactFlags
}

// NewContactHeader creates a new ContactHeader with given ContactFlags.
func NewContactHeader(flags ContactFlags) ContactHeader {
	return ContactHeader{
		Flags: flags,
	}
}

func (ch ContactHeader) String() string {
	return fmt.Sprintf("ContactHeader(Version=4, Flags=%v)", ch.Flags)
}

// MarshalBinary encodes this ContactHeader into its binary form.
func (ch ContactHeader) MarshalBinary() (data []byte, _ error) {
	// magic='dtn!', version=4, flags=flags
	data = []byte{0x64, 0x74, 0x6E, 0x21, 0x04, byte(ch.Flags)}
	return
}

// UnmarshalBinary decodes a ContactHeader from its binary form.
func (ch *ContactHeader) UnmarshalBinary(data []byte) error {
	if len(data) != 6 {
		return fmt.Errorf("ContactHeader's length is wrong: %d instead of 6", len(data))
	}

	if !bytes.Equal(data[:4], []byte("dtn!")) {
		return fmt.Errorf("ContactHeader's magic does not match: %x != 'dtn!'", data[:4])
	}

	if uint8(data[4]) != 4 {
		return fmt.Errorf("ContactHeader's version is wrong: %d instead of 4", uint8(data[4]))
	}

	if cf := ContactFlags(data[5]); cf&contactFlags_INVALID != 0 {
		return fmt.Errorf("ContactHeader's flags %x contain invalid flags", cf)
	} else {
		ch.Flags = cf
	}

	return nil
}
