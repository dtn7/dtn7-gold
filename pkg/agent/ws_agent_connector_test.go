// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestWebAgentConnector(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start WebSocketAgent server
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	ws := NewWebSocketAgent()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/ws", ws.ServeHTTP)
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

	// Attach Connector
	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   "/ws",
	}
	wac, wacErr := NewWebSocketAgentConnector(u.String(), "dtn://foobar/23")
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
		t.Fatal("WebSocketAgent did not received message; time out")
	}

	b = createBundle("dtn://server/", "dtn://foobar/23", t)
	ws.MessageReceiver() <- BundleMessage{b}

	time.Sleep(250 * time.Millisecond)

	fin := make(chan struct{})
	go func() {
		if b2, err := wac.ReadBundle(); err != nil {
			return
		} else if !reflect.DeepEqual(b, b2) {
			return
		}

		// fin will only be closed iff no error occurred. Otherwise the timeout below will hit.
		close(fin)
	}()

	select {
	case <-fin:
		break
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	go func() {
		msg := <-ws.MessageSender()
		if msg, ok := msg.(SyscallRequestMessage); !ok {
			ws.MessageReceiver() <- SyscallResponseMessage{
				Request:   msg.Request,
				Response:  []byte{},
				Recipient: msg.Sender,
			}
		} else {
			ws.MessageReceiver() <- SyscallResponseMessage{
				Request:   msg.Request,
				Response:  []byte{0xAC, 0xAB},
				Recipient: msg.Sender,
			}
		}
	}()

	if response, err := wac.Syscall("test", time.Millisecond); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(response, []byte{0xAC, 0xAB}) {
		t.Fatalf("received %x", response)
	}

	wac.Close()

	// Let the WebSocketAgent act on the closed connection
	time.Sleep(250 * time.Millisecond)

	ws.MessageReceiver() <- ShutdownMessage{}

	// Let the WebSocketAgent shut itself down
	time.Sleep(250 * time.Millisecond)
}
