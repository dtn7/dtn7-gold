package tcpcl

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// SESS_INIT is the Message Header code for a Session Initialization Message.
const SESS_INIT uint8 = 0x07

// SessionInitMessage is the SESS_INIT message to negotiate session parameters.
type SessionInitMessage struct {
	KeepaliveInterval uint16
	SegmentMru        uint64
	TransferMru       uint64
	Eid               string

	// TODO: Session Extension Items
}

// NewSessionInitMessage creates a new SessionInitMessage with given fields.
func NewSessionInitMessage(keepaliveInterval uint16, segmentMru, transferMru uint64, eid string) SessionInitMessage {
	return SessionInitMessage{
		KeepaliveInterval: keepaliveInterval,
		SegmentMru:        segmentMru,
		TransferMru:       transferMru,
		Eid:               eid,
	}
}

func (si SessionInitMessage) String() string {
	return fmt.Sprintf(
		"SESS_INIT(Keepalive Interval=%d, Segment MRU=%d, Transfer MRU=%d, EID=%s)",
		si.KeepaliveInterval, si.SegmentMru, si.TransferMru, si.Eid)
}

// MarshalBinary encodes this SessionInitMessage into its binary form.
func (si SessionInitMessage) MarshalBinary() (data []byte, err error) {
	var buf = new(bytes.Buffer)
	var fields = []interface{}{
		SESS_INIT,
		si.KeepaliveInterval,
		si.SegmentMru,
		si.TransferMru,
		uint16(len(si.Eid))}

	for _, field := range fields {
		if binErr := binary.Write(buf, binary.BigEndian, field); binErr != nil {
			err = binErr
			return
		}
	}

	if n, _ := buf.WriteString(si.Eid); n != len(si.Eid) {
		err = fmt.Errorf("SESS_INIT EID's length is %d, but only wrote %d bytes", len(si.Eid), n)
		return
	}

	// TODO: Session Extension Items
	// Currently, only an empty Session Extension Items Length is accepted.
	if binErr := binary.Write(buf, binary.BigEndian, uint32(0)); binErr != nil {
		err = binErr
		return
	}

	data = buf.Bytes()
	return
}

// UnmarshalBinary decodes a SessionInitMessage from its binary form.
func (si *SessionInitMessage) UnmarshalBinary(data []byte) error {
	var buf = bytes.NewReader(data)

	var messageHeader uint8
	if err := binary.Read(buf, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != SESS_INIT {
		return fmt.Errorf("SESS_INIT's Message Header is wrong: %d instead of %d", messageHeader, SESS_INIT)
	}

	var eidLength uint16
	var fields = []interface{}{
		&si.KeepaliveInterval,
		&si.SegmentMru,
		&si.TransferMru,
		&eidLength,
	}

	for _, field := range fields {
		if err := binary.Read(buf, binary.BigEndian, field); err != nil {
			return err
		}
	}

	var eidBuff = make([]byte, eidLength)
	if n, err := buf.Read(eidBuff); err != nil {
		return err
	} else if uint16(n) != eidLength {
		return fmt.Errorf("SESS_INIT's EID length differs: expected %d and got %d", eidLength, n)
	} else {
		si.Eid = string(eidBuff)
	}

	// TODO: Session Extension Items, see above
	var sessionExtsLen uint32
	if err := binary.Read(buf, binary.BigEndian, &sessionExtsLen); err != nil {
		return err
	} else if sessionExtsLen > 0 {
		sessionExtsBuff := make([]byte, sessionExtsLen)

		if n, err := buf.Read(sessionExtsBuff); err != nil {
			return err
		} else if uint32(n) != sessionExtsLen {
			return fmt.Errorf(
				"SESS_INIT's Session Extension Length  differs: expected %d and got %d",
				sessionExtsLen, n)
		}
	}

	if n := buf.Len(); n > 0 {
		return fmt.Errorf("SESS_INIT's buffer should be empty; has %d octets", n)
	}

	return nil
}
