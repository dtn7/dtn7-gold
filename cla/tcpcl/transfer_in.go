package tcpcl

import (
	"bytes"
	"fmt"
	"io"
)

type IncomingTransfer struct {
	Id uint64

	endFlag bool
	buf     *bytes.Buffer
}

func NewIncomingTransfer(id uint64) *IncomingTransfer {
	return &IncomingTransfer{
		Id:  id,
		buf: new(bytes.Buffer),
	}
}

func (t IncomingTransfer) String() string {
	return fmt.Sprintf("INCOMING_TRANSFER(%d)", t.Id)
}

func (t *IncomingTransfer) NextSegment(dtm DataTransmissionMessage) (dam DataAcknowledgementMessage, err error) {
	if t.endFlag {
		err = fmt.Errorf("Transfer has already received an end flag")
		return
	}

	if t.Id != dtm.TransferId {
		err = fmt.Errorf("XFER_SEGMENT's Transfer ID %d mismatches %d", dtm.TransferId, t.Id)
		return
	}

	dtmReader := bytes.NewBuffer(dtm.Data)
	if _, cpyErr := io.Copy(t.buf, dtmReader); cpyErr != nil {
		err = cpyErr
		return
	}

	if dtm.Flags&SegmentEnd != 0 {
		t.endFlag = true
	}

	dam = NewDataAcknowledgementMessage(dtm.Flags, dtm.TransferId, uint64(t.buf.Len()))
	return
}

func (t *IncomingTransfer) IsFinished() bool {
	return t.endFlag
}
