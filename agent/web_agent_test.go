package agent

import (
	"fmt"
	"net"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// randomPort returns a random open TCP port.
func randomPort(t *testing.T) (port int) {
	if addr, err := net.ResolveTCPAddr("tcp", "localhost:0"); err != nil {
		t.Fatal(err)
	} else if l, err := net.ListenTCP("tcp", addr); err != nil {
		t.Fatal(err)
	} else {
		port = l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
	}
	return
}

// isAddrReachable checks if a TCP address - like localhost:2342 - is reachable.
func isAddrReachable(addr string) (open bool) {
	if conn, err := net.DialTimeout("tcp", addr, time.Second); err != nil {
		open = false
	} else {
		open = true
		_ = conn.Close()
	}
	return
}

func TestWebAgentNew(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start WebAgent server
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

	// Connect dummy client
	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   "/ws",
	}
	wsClient, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Register client
	if w, err := wsClient.NextWriter(websocket.BinaryMessage); err != nil {
		t.Fatal(err)
	} else if err := marshalCbor(newRegisterMessage("dtn://foobar/"), w); err != nil {
		t.Fatal(err)
	} else if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Check registration
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

	// Send Bundle to client
	b, err := bundle.Builder().
		Source("dtn:test").
		Destination("dtn://foobar/").
		CreationTimestampNow().
		Lifetime("24h").
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	ws.MessageReceiver() <- BundleMessage{b}

	// Check received Bundle
	if mt, r, err := wsClient.NextReader(); err != nil {
		t.Fatal(err)
	} else if mt != websocket.BinaryMessage {
		t.Fatalf("expected message type %v, got %v", websocket.BinaryMessage, mt)
	} else if msg, err := unmarshalCbor(r); err != nil {
		t.Fatal(err)
	} else if msg.typeCode() != wamBundleCode {
		t.Fatalf("expected status code %d, got %d", wamBundleCode, msg.typeCode())
	} else if bRecv := msg.(*wamBundle).b; !reflect.DeepEqual(b, bRecv) {
		t.Fatalf("expected %v, got %v", b, bRecv)
	}

	// Shutdown WebAgent with all its child processes
	ws.MessageReceiver() <- ShutdownMessage{}

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
