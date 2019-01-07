package bundle

import (
	"fmt"
	"time"
)

// DtnTime is an integer indicating the time like the Unix time, just starting
// from the year 2000 instead of 1970. It is specified in section 4.1.6.
type DtnTime uint

const (
	seconds1970To2k = 946684800

	// DtnTimeEpoch represents the zero timestamp/epoch.
	DtnTimeEpoch DtnTime = 0
)

// Unix returns the Unix timestamp for this DtnTime.
func (t DtnTime) Unix() int64 {
	return int64(t) + seconds1970To2k
}

// Time returns a UTC-based time.Time for this DtnTime.
func (t DtnTime) Time() time.Time {
	return time.Unix(t.Unix(), 0).UTC()
}

// DtnTimeFromTime returns the DtnTime for the time.Time.
func DtnTimeFromTime(t time.Time) DtnTime {
	return (DtnTime)(t.Unix() - seconds1970To2k)
}

// DtnTimeNow returns the current (UTC) time as DtnTime.
func DtnTimeNow() DtnTime {
	return DtnTimeFromTime(time.Now())
}

// CreationTimestamp is a tuple of a DtnTime and a sequence number (to differ
// bundles with the same DtnTime (seconds) from the same endpoint). It is
// specified in section 4.1.7.
type CreationTimestamp [2]uint

// NewCreationTimestamp creates a new creation timestamp from a given DTN time
// and a sequence number, resulting in a hopefully unique tuple.
func NewCreationTimestamp(time DtnTime, sequence uint) CreationTimestamp {
	return [2]uint{uint(time), sequence}
}

// DtnTime returns the creation timestamp's DTN time part.
func (ct CreationTimestamp) DtnTime() DtnTime {
	return DtnTime(ct[0])
}

// SequenceNumber returns the creation timestamp's sequence number.
func (ct CreationTimestamp) SequenceNumber() uint {
	return ct[1]
}

func (ct CreationTimestamp) String() string {
	return fmt.Sprintf("(%d, %d)", ct[0], ct[1])
}
