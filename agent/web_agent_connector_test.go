package agent

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestWebAgentConnector(t *testing.T) {
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

	// Attach Connector
	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   "/ws",
	}
	wac, wacErr := NewWebAgentConnector(u.String(), "dtn://foobar/23")
	if wacErr != nil {
		t.Fatal(wacErr)
	}

	b := createBundle("dtn://foobar/23", "dtn://server/", t)
	if err := wac.WriteBundle(b); err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)

	select {
	case msg := <-ws.MessageSender():
		if bMsg, ok := msg.(BundleMessage); !ok {
			t.Fatalf("expected BundleMessage, got %T", msg)
		} else if !reflect.DeepEqual(b, bMsg.Bundle) {
			t.Fatalf("expected %v, got %v", b, bMsg.Bundle)
		}

	case <-time.After(500 * time.Millisecond):
		t.Fatal("WebAgent did not received message; time out")
	}

	b = createBundle("dtn://server/", "dtn://foobar/23", t)
	ws.MessageReceiver() <- BundleMessage{b}

	time.Sleep(250 * time.Millisecond)

	fin := make(chan struct{})
	go func() {
		if b2, err := wac.ReadBundle(); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(b, b2) {
			t.Fatalf("expected %v, got %v", b, b2)
		}
		close(fin)
	}()

	select {
	case <-fin:
		break
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	wac.Close()

	// Let the WebAgent act on the closed connection
	time.Sleep(250 * time.Millisecond)

	ws.MessageReceiver() <- ShutdownMessage{}

	// Let the WebAgent shut itself down
	time.Sleep(250 * time.Millisecond)
}
