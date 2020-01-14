package agent

import (
	"bytes"
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
	if wsClient, _, err := websocket.DefaultDialer.Dial(u.String(), nil); err != nil {
		t.Fatal(err)
	} else if mt, p, err := wsClient.ReadMessage(); err != nil {
		t.Fatal(err)
	} else if mt != websocket.TextMessage {
		t.Fatalf("Message Type is %d, not %d", mt, websocket.TextMessage)
	} else if !bytes.Equal(p, []byte("GuMo")) {
		t.Fatalf("Payload was %x / %s", p, p)
	} else if err := wsClient.Close(); err != nil {
		t.Fatal(err)
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
