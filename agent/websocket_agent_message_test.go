package agent

import (
	"bytes"
	"reflect"
	"testing"
)

func TestWebsocketAgentMessageEnDecode(t *testing.T) {
	msgs := []websocketAgentMessage{
		&wamRegister{"dtn:foobar"},
	}

	for _, msg := range msgs {
		var buff bytes.Buffer

		if err := marshalWam(msg, &buff); err != nil {
			t.Fatal(err)
		}

		if msg2, err := unmarshalWam(&buff); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(msg, msg2) {
			t.Fatalf("Messages differ: %v %v", msg, msg2)
		}
	}
}
