// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/dtn7/cboring"
)

// DtnTime is an integer representation of milliseconds since the start of the year 2000 (UTC).
type DtnTime uint64

const (
	milliseconds1970To2k = 946684800000

	milliToSec  int64 = 1000
	nanoToMilli int64 = 1000000

	// DtnTimeEpoch represents the zero timestamp/epoch.
	DtnTimeEpoch DtnTime = 0
)

// unixMilliseconds returns the DntTime's milliseconds since Unix epoch.
func (t DtnTime) unixMilliseconds() int64 {
	return int64(t) + milliseconds1970To2k
}

// Time returns a UTC-based time.Time for this DtnTime.
func (t DtnTime) Time() time.Time {
	unixSec := t.unixMilliseconds() / milliToSec
	unixNano := (t.unixMilliseconds() - (unixSec * milliToSec)) * nanoToMilli

	return time.Unix(unixSec, unixNano).UTC()
}

// String returns this DtnTime's string representation.
func (t DtnTime) String() string {
	return t.Time().Format("2006-01-02 15:04:05.000")
}

// DtnTimeFromTime returns the DtnTime for the time.Time.
func DtnTimeFromTime(t time.Time) DtnTime {
	return (DtnTime)((t.UTC().UnixNano() / nanoToMilli) - milliseconds1970To2k)
}

// DtnTimeNow returns the current (UTC) time as DtnTime.
func DtnTimeNow() DtnTime {
	return DtnTimeFromTime(time.Now())
}

// CreationTimestamp is a tuple of a DtnTime and a sequence number (to differ
// bundles with the same DtnTime (seconds) from the same endpoint). It is
// specified in section 4.1.7.
type CreationTimestamp [2]uint64

// NewCreationTimestamp creates a new creation timestamp from a given DTN time
// and a sequence number, resulting in a hopefully unique tuple.
func NewCreationTimestamp(time DtnTime, sequence uint64) CreationTimestamp {
	return [2]uint64{uint64(time), sequence}
}

// DtnTime returns the creation timestamp's DTN time part.
func (ct CreationTimestamp) DtnTime() DtnTime {
	return DtnTime(ct[0])
}

// IsZeroTime returns if the time part is set to zero, indicating the lack of
// an accurate clock.
func (ct CreationTimestamp) IsZeroTime() bool {
	return ct.DtnTime() == DtnTimeEpoch
}

// SequenceNumber returns the creation timestamp's sequence number.
func (ct CreationTimestamp) SequenceNumber() uint64 {
	return ct[1]
}

func (ct CreationTimestamp) String() string {
	return fmt.Sprintf("(%v, %d)", DtnTime(ct[0]), ct[1])
}

// MarshalCbor writes a CBOR representation for this CreationTimestamp.
func (ct *CreationTimestamp) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	for _, f := range ct {
		if err := cboring.WriteUInt(f, w); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalCbor reads a CBOR representation of a CreationTimestamp.
func (ct *CreationTimestamp) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("expected array with length 2, got %d", l)
	}

	for i := 0; i < 2; i++ {
		if f, err := cboring.ReadUInt(r); err != nil {
			return err
		} else {
			ct[i] = f
		}
	}

	return nil
}

// MarshalJSON creates a JSON object representing this CreationTimestamp.
func (ct CreationTimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Date string `json:"date"`
		Seq  uint64 `json:"sequenceNo"`
	}{
		Date: ct.DtnTime().String(),
		Seq:  ct.SequenceNumber(),
	})
}
