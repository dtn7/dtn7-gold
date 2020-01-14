package agent

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebAgentNew(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	ws, wsErr := NewWebAgent(addr)
	if wsErr != nil {
		t.Fatal(wsErr)
	}

	// Let the WebAgent start..
	time.Sleep(250 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		if isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("SocketAgent seems to be unreachable")
		}
	}

	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   "/ws",
	}
	wsClient, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if w, err := wsClient.NextWriter(websocket.BinaryMessage); err != nil {
		t.Fatal(err)
	} else if err := marshalCbor(newRegisterMessage("dtn:foobar"), w); err != nil {
		t.Fatal(err)
	} else if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if mt, r, err := wsClient.NextReader(); err != nil {
		t.Fatal(err)
	} else if mt != websocket.BinaryMessage {
		t.Fatalf("expected message type %v, got %v", websocket.BinaryMessage, mt)
	} else if msg, err := unmarshalCbor(r); err != nil {
		t.Fatal(err)
	} else if msg.typeCode() != wamStatusCode {
		t.Fatalf("expected status code %d, got %d", wamStatusCode, msg.typeCode())
	} else if msg := msg.(*wamStatus); msg.errorMsg != "" {
		t.Fatal(msg.errorMsg)
	}

	ws.receiver <- ShutdownMessage{}

	// Let the WebAgent shut itself down
	time.Sleep(250 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		if !isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("WebAgent is still reachable")
		}
	}

}
