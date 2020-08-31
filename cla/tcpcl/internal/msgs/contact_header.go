// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"fmt"
	"io"
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

// contactHeaderHead represents both the magic "dtn!" and the version field (4).
var contactHeaderHead = []byte{0x64, 0x74, 0x6E, 0x21, 0x04}

// NewContactHeader creates a new ContactHeader with given ContactFlags.
func NewContactHeader(flags ContactFlags) *ContactHeader {
	return &ContactHeader{
		Flags: flags,
	}
}

func (ch ContactHeader) String() string {
	return fmt.Sprintf("ContactHeader(Version=4, Flags=%v)", ch.Flags)
}

func (ch ContactHeader) Marshal(w io.Writer) error {
	var data = append(contactHeaderHead, byte(ch.Flags))

	if n, err := w.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("wrote %d octets instead of %d", n, len(data))
	}

	return nil
}

func (ch *ContactHeader) Unmarshal(r io.Reader) error {
	var data = make([]byte, 6)

	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}

	if head := data[:5]; !bytes.Equal(head, contactHeaderHead) {
		return fmt.Errorf("ContactHeader's magic and version does not match: %x != %x", head, contactHeaderHead)
	}

	ch.Flags = ContactFlags(data[5])

	return nil
}
