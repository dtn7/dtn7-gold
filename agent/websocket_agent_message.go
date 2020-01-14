package agent

import (
	"fmt"
	"io"
	"reflect"

	"github.com/dtn7/cboring"
)

// websocketAgentMessage describes a message which might be sent over a WebsocketAgent.
// Implementations are available at the end of this file.
type websocketAgentMessage interface {
	// typeCode is an unique identifier for each message type.
	// A const list of those and a map to a specific type will follow this interface's definition.
	typeCode() uint64

	// CborMarshaler must only be implemented for the type's logic. A generic wrapper for the typeCode is available
	// in the marshalWam and unmarshalWam functions.
	cboring.CborMarshaler
}

const wamRegisterCode uint64 = 0

var wamCodes = map[uint64]reflect.Type{
	wamRegisterCode: reflect.TypeOf(wamRegister{}),
}

// marshalWam writes a websocketAgentMessage wrapped with its type code.
func marshalWam(wam websocketAgentMessage, w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	if err := cboring.WriteUInt(wam.typeCode(), w); err != nil {
		return err
	}

	if err := cboring.Marshal(wam, w); err != nil {
		return err
	}

	return nil
}

// unmarshalWam reads a new websocketAgentMessage based on its type code.
func unmarshalWam(r io.Reader) (wam websocketAgentMessage, err error) {
	if n, arrErr := cboring.ReadArrayLength(r); arrErr != nil {
		err = arrErr
		return
	} else if n != 2 {
		err = fmt.Errorf("expected array of two elements, got %d", n)
		return
	}

	if n, typeErr := cboring.ReadUInt(r); typeErr != nil {
		err = typeErr
		return
	} else if t, ok := wamCodes[n]; !ok {
		err = fmt.Errorf("no known WAM type code %d", n)
		return
	} else {
		wam = reflect.New(t).Interface().(websocketAgentMessage)
	}

	if wamErr := cboring.Unmarshal(wam, r); wamErr != nil {
		err = wamErr
		return
	}

	return
}

type wamRegister struct {
	endpoint string
}

func (_ *wamRegister) typeCode() uint64 {
	return wamRegisterCode
}

func (wr *wamRegister) MarshalCbor(w io.Writer) error {
	return cboring.WriteTextString(wr.endpoint, w)
}

func (wr *wamRegister) UnmarshalCbor(r io.Reader) (err error) {
	wr.endpoint, err = cboring.ReadTextString(r)
	return
}
