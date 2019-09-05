package tcpcl

import (
	"bytes"
	"fmt"
	"strings"
)

// ContactFlags are single-bit flags used in the ContactHeader.
type ContactFlags uint8

const (
	// ContactCanTls indicates that the sending peer is capable of TLS security.
	ContactCanTls ContactFlags = 0x01
)

func (cf ContactFlags) String() string {
	var flags []string

	if cf&ContactCanTls != 0 {
		flags = append(flags, "CAN_TLS")
	}

	return strings.Join(flags, ",")
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

	ch.Flags = ContactFlags(data[5])

	return nil
}
