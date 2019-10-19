package bbc

// Fragment is a part of a Transmission. Multiple Fragments represent an entire Transmission.
//
// For identification, a tuple consisting of a transmission ID, a sequence number, a start and an end
// bit is used. Because of memory reasons, the whole identifier has a size of one byte. First the
// transmission ID with a length of four bits is given, followed by the two bit long sequence number and
// the two start and end bits.
//
// The transmission ID is taken from the respective Transmission. If this Fragment is the first of a
// Transmission, the start bit is set to one. The same applies to the end bit for the last Fragment.
//
// The two bit sequence number represents a simple binary counter, which is incremented for each Fragment.
// Missing Fragments can thus be detected. Due to the length of two bits, a multiple of four Fragments must
// be lost so that an error is not recognized. This seems less likely than an error detection by just one
// alternating bit.
//
// After the one byte identifier the payload of this Fragment follows. This can be as long as the particular
// MTU will allow.
//
//     0   1   2   3   4   5   6   7
//   +---+---+---+---+---+---+---+---+
//   |Transmission ID|Seq. No|SB |EB |
//   +---+---+---+---+---+---+---+---+
//   |                               |
//   +            Payload            +
//   |                               |
//
type Fragment struct {
	identifier byte
	Payload    []byte
}

// fragmentIdentifierSize is the additional size for each Fragment's header.
const fragmentIdentifierSize int = 1

// NewFragment creates a new Fragment based on the given arguments.
func NewFragment(transmissionID, sequenceNo byte, start, end bool, payload []byte) Fragment {
	var identifier byte = 0x00
	identifier |= (transmissionID & 0x0F) << 4
	identifier |= (sequenceNo & 0x03) << 2
	if start {
		identifier |= 0x02
	}
	if end {
		identifier |= 0x01
	}

	return Fragment{
		identifier: identifier,
		Payload:    payload,
	}
}

// TransmissionID returns the four bit transmission ID.
func (f Fragment) TransmissionID() byte {
	return f.identifier >> 4 & 0x0F
}

// SequenceNumber returns the two bit sequence number.
func (f Fragment) SequenceNumber() byte {
	return f.identifier >> 2 & 0x03
}

// StartBit checks if the start bit is set.
func (f Fragment) StartBit() bool {
	return f.identifier&0x02 != 0
}

// EndBit checks if the end bit is set.
func (f Fragment) EndBit() bool {
	return f.identifier&0x01 != 0
}

// nextSequenceNumber returns the succeeding sequence number.
func nextSequenceNumber(seq byte) byte {
	return (seq + 1) % 4
}
