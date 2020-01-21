package agent

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
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

// createBundle from src to dst for testing purpose.
func createBundle(src, dst string, t *testing.T) bundle.Bundle {
	b, err := bundle.Builder().
		Source(src).
		Destination(dst).
		CreationTimestampNow().
		Lifetime("24h").
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestWebAgentNew(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start WebSocketAgent server
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	ws := NewWebSocketAgent()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/ws", ws.WebsocketHandler)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: httpMux,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	// Let the WebSocketAgent start..
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
	b := createBundle("dtn://test/", "dtn://foobar/", t)
	ws.MessageReceiver() <- BundleMessage{b}

	// Client checks received Bundle
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

	// Send Bundle from client
	b = createBundle("dtn://foobar/", "dtn://test/", t)
	if w, err := wsClient.NextWriter(websocket.BinaryMessage); err != nil {
		t.Fatal(err)
	} else if err := marshalCbor(newBundleMessage(b), w); err != nil {
		t.Fatal(err)
	} else if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Server checks received Bundle
	select {
	case msg := <-ws.MessageSender():
		if msg, ok := msg.(BundleMessage); !ok {
			t.Fatalf("Message is not a Bundle Message; %v", msg)
		} else if !reflect.DeepEqual(b, msg.Bundle) {
			t.Fatalf("expected %v, got %v", b, msg.Bundle)
		}

	case <-time.After(500 * time.Millisecond):
		t.Fatal("Bundle reception timed out")
	}

	// Send syscall request from client
	if w, err := wsClient.NextWriter(websocket.BinaryMessage); err != nil {
		t.Fatal(err)
	} else if err := marshalCbor(newSyscallRequestMessage("test"), w); err != nil {
		t.Fatal(err)
	} else if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Server responds to the syscall
	select {
	case msg := <-ws.MessageSender():
		if msg, ok := msg.(SyscallRequestMessage); !ok {
			t.Fatalf("Message is not a SyscallRequestMessage; %v", msg)
		} else if msg.Request != "test" {
			t.Fatalf("expected payload of \"test\", not %s", msg.Request)
		} else {
			ws.MessageReceiver() <- SyscallResponseMessage{
				Request:   "test",
				Response:  []byte{0x23, 0x42},
				Recipient: bundle.MustNewEndpointID("dtn://foobar/"),
			}
		}

	case <-time.After(500 * time.Millisecond):
		t.Fatal("syscall request reception timed out")
	}

	// Client checks the syscall response
	if mt, r, err := wsClient.NextReader(); err != nil {
		t.Fatal(err)
	} else if mt != websocket.BinaryMessage {
		t.Fatalf("expected message type %v, got %v", websocket.BinaryMessage, mt)
	} else if msg, err := unmarshalCbor(r); err != nil {
		t.Fatal(err)
	} else if msg.typeCode() != wamSyscallResponseCode {
		t.Fatalf("expected status code %d, got %d", wamSyscallResponseCode, msg.typeCode())
	} else if response := msg.(*wamSyscallResponse).response; !bytes.Equal(response, []byte{0x23, 0x42}) {
		t.Fatalf("received %x", response)
	}

	// Shutdown WebSocketAgent with all its child processes
	ws.MessageReceiver() <- ShutdownMessage{}
}

func TestWebAgentIllegalEndpoint(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start WebSocketAgent server
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	ws := NewWebSocketAgent()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/ws", ws.WebsocketHandler)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: httpMux,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	// Let the WebSocketAgent start..
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

	// Register client with an illegal endpoint ID
	if w, err := wsClient.NextWriter(websocket.BinaryMessage); err != nil {
		t.Fatal(err)
	} else if err := marshalCbor(newRegisterMessage("uff"), w); err != nil {
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
	} else if msg := msg.(*wamStatus); msg.errorMsg == "" {
		t.Fatal("Expected error due to illegal endpoint ID")
	}

	// Shutdown WebSocketAgent
	ws.MessageReceiver() <- ShutdownMessage{}
}
