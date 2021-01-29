// SPDX-FileCopyrightText: 2020 Markus Sommer
// SPDX-FileCopyrightText: 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package discovery

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
)

// Announcement of some node's CLA.
type Announcement struct {
	Type     cla.CLAType
	Endpoint bpv7.EndpointID
	Port     uint
}

// UnmarshalAnnouncements creates a new array of Announcement based on a CBOR byte string.
func UnmarshalAnnouncements(data []byte) (announcements []Announcement, err error) {
	buff := bytes.NewBuffer(data)

	if l, cErr := cboring.ReadArrayLength(buff); cErr != nil {
		err = cErr
		return
	} else {
		announcements = make([]Announcement, l)
	}

	for i := 0; i < len(announcements); i++ {
		if cErr := cboring.Unmarshal(&announcements[i], buff); cErr != nil {
			err = fmt.Errorf("unmarshalling Announcement %d failed: %v", i, cErr)
			return
		}
	}

	return
}

// MarshalAnnouncements into a CBOR byte string.
func MarshalAnnouncements(announcements []Announcement) (data []byte, err error) {
	buff := new(bytes.Buffer)

	if cErr := cboring.WriteArrayLength(uint64(len(announcements)), buff); cErr != nil {
		err = cErr
		return
	}

	for i := range announcements {
		// Don't "range" variable because gosec's G601: Implicit memory aliasing in for loop.
		announcement := announcements[i]
		if cErr := cboring.Marshal(&announcement, buff); cErr != nil {
			err = fmt.Errorf("marshalling Announcement %d (%v) failed: %v", i, announcement, cErr)
			return
		}
	}

	data = buff.Bytes()
	return
}

// MarshalCbor creates a CBOR representation for an Announcement.
func (announcement *Announcement) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(3, w); err != nil {
		return err
	}

	if err := cboring.WriteUInt(uint64(announcement.Type), w); err != nil {
		return err
	}
	if err := cboring.Marshal(&announcement.Endpoint, w); err != nil {
		return fmt.Errorf("marshalling endpoint failed: %v", err)
	}
	if err := cboring.WriteUInt(uint64(announcement.Port), w); err != nil {
		return err
	}

	return nil
}

// UnmarshalCbor creates an Announcement from its CBOR representation.
func (announcement *Announcement) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 3 {
		return fmt.Errorf("wrong array length: %d instead of 3", l)
	}

	if n, err := cboring.ReadUInt(r); err != nil {
		return err
	} else if claType := cla.CLAType(n); claType.CheckValid() != nil {
		return claType.CheckValid()
	} else {
		announcement.Type = claType
	}
	if err := cboring.Unmarshal(&announcement.Endpoint, r); err != nil {
		return fmt.Errorf("unmarshalling endpoint failed: %v", err)
	}
	if n, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		announcement.Port = uint(n)
	}

	return nil
}

func (announcement Announcement) String() string {
	return fmt.Sprintf("Announcement(%v,%v,%d)", announcement.Type, announcement.Endpoint, announcement.Port)
}
