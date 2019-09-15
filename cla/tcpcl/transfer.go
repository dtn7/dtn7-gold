package tcpcl

import (
	"bufio"
	"io"

	"github.com/dtn7/dtn7-go/bundle"
)

// Transfer represents a Bundle Transfer for the TCPCL.
type Transfer struct {
	Id uint64

	startFlag  bool
	dataStream io.Reader
}

// NewTransfer creates a new Transfer for data written into the returned Writer.
func NewTransfer(id uint64) (t *Transfer, w io.Writer) {
	r, w := io.Pipe()
	t = &Transfer{
		Id:         id,
		startFlag:  true,
		dataStream: r,
	}

	return
}

// NewBundleTransfer creates a new Transfer for a Bundle.
func NewBundleTransfer(id uint64, b bundle.Bundle) *Transfer {
	var t, w = NewTransfer(id)

	go func(w *io.PipeWriter) {
		bw := bufio.NewWriter(w)

		_ = b.MarshalCbor(bw)
		_ = bw.Flush()
		_ = w.Close()
	}(w.(*io.PipeWriter))

	return t
}

// NextSegment creates the next XFER_SEGMENT for the given MRU or an EOF in case
// of a finished Writer.
func (t *Transfer) NextSegment(mru uint64) (dtm DataTransmissionMessage, err error) {
	var segFlags SegmentFlags

	if t.startFlag {
		t.startFlag = false
		segFlags |= SegmentStart
	}

	var buf = make([]byte, mru)
	if n, rErr := t.dataStream.Read(buf); rErr != nil {
		err = rErr
		return
	} else if uint64(n) < mru {
		segFlags |= SegmentEnd
	}

	dtm = NewDataTransmissionMessage(segFlags, t.Id, buf)
	return
}
