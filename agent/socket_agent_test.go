package agent

import (
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
)

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

func isAddrReachable(addr string) (open bool) {
	if conn, err := net.DialTimeout("tcp", addr, time.Second); err != nil {
		open = false
	} else {
		open = true
		_ = conn.Close()
	}
	return
}

func TestSocketAgentNew(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	socket, socketErr := NewSocket(addr, bundle.MustNewEndpointID("dtn://foo/bar"))
	if socketErr != nil {
		t.Fatal(socketErr)
	}

	// Let the SocketAgent start..
	time.Sleep(250 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		if isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("SocketAgent seems to be unreachable")
		}
	}

	socket.receiver <- ShutdownMessage{}

	// Let the SocketAgent shut itself down
	time.Sleep(250 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		if !isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("SocketAgent is still reachable")
		}
	}
}

func TestSocketAgentReceive(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	socket, socketErr := NewSocket(addr, bundle.MustNewEndpointID("dtn://foo/bar"))
	if socketErr != nil {
		t.Fatal(socketErr)
	}

	// Let the SocketAgent start..
	time.Sleep(250 * time.Millisecond)

	var conn net.Conn
	for i := 1; i <= 3; i++ {
		c, cErr := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if cErr == nil {
			conn = c
			break
		} else if i == 3 {
			t.Fatal(cErr)
		}
	}
	if conn == nil {
		t.Fatal("conn is nil, wat")
	}

	bndl1, bndlErr := bundle.Builder().
		Source("dtn://oof/").
		Destination("dtn://foo/bar").
		CreationTimestampNow().
		Lifetime("1h").
		PayloadBlock([]byte("hello")).
		Build()

	if bndlErr != nil {
		t.Fatal(bndlErr)
	}

	var bndl2 bundle.Bundle

	fin := make(chan struct{})
	go func() {
		if err := bndl2.UnmarshalCbor(conn); err != nil {
			t.Fatal(err)
		}
		close(fin)
	}()

	time.Sleep(250 * time.Millisecond)
	socket.receiver <- BundleMessage{bndl1}

	select {
	case <-fin:
		break
	case <-time.After(3 * time.Second):
		t.Fatal("Unmarshalling CBOR timed out")
	}

	if !reflect.DeepEqual(bndl1, bndl2) {
		t.Fatal("Bundles differ")
	}

	_ = conn.Close()
	socket.receiver <- ShutdownMessage{}
}

func TestSocketAgentSend(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	socket, socketErr := NewSocket(addr, bundle.MustNewEndpointID("dtn://foo/bar"))
	if socketErr != nil {
		t.Fatal(socketErr)
	}

	// Let the SocketAgent start..
	time.Sleep(250 * time.Millisecond)

	conn, connErr := net.DialTimeout("tcp", addr, time.Second)
	if connErr != nil {
		t.Fatal(connErr)
	}

	bndl, bndlErr := bundle.Builder().
		Source("dtn://foo/bar").
		Destination("dtn://oof/").
		CreationTimestampNow().
		Lifetime("1h").
		PayloadBlock([]byte("hello")).
		Build()

	if bndlErr != nil {
		t.Fatal(bndlErr)
	}
	if err := bndl.MarshalCbor(conn); err != nil {
		t.Fatal(err)
	}

	msg := <-socket.sender
	if bMsg, ok := msg.(BundleMessage); !ok {
		t.Fatalf("Message was not a BundleMessage; %T", msg)
	} else if !reflect.DeepEqual(bndl, bMsg.Bundle) {
		t.Fatal("Bundles differ")
	}

	_ = conn.Close()
	socket.receiver <- ShutdownMessage{}
}
