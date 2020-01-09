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
	if conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err != nil {
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

	// Let the Socket start..
	time.Sleep(100 * time.Millisecond)

	if !isAddrReachable(addr) {
		t.Fatal("Socket seems to be unreachable")
	}

	socket.receiver <- ShutdownMessage{}

	// Let the Socket shut itself down
	time.Sleep(100 * time.Millisecond)

	if isAddrReachable(addr) {
		t.Fatal("Socket is still reachable..")
	}
}

func TestSocketAgentReceive(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	socket, socketErr := NewSocket(addr, bundle.MustNewEndpointID("dtn://foo/bar"))
	if socketErr != nil {
		t.Fatal(socketErr)
	}

	// Let the Socket start..
	time.Sleep(100 * time.Millisecond)

	conn, connErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if connErr != nil {
		t.Fatal(connErr)
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

	socket.receiver <- BundleMessage{bndl1}

	var bndl2 bundle.Bundle
	if err := bndl2.UnmarshalCbor(conn); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bndl1, bndl2) {
		t.Fatal("Bundles differ")
	}

	socket.receiver <- ShutdownMessage{}
}

func TestSocketAgentSend(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	socket, socketErr := NewSocket(addr, bundle.MustNewEndpointID("dtn://foo/bar"))
	if socketErr != nil {
		t.Fatal(socketErr)
	}

	// Let the Socket start..
	time.Sleep(100 * time.Millisecond)

	conn, connErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
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

	socket.receiver <- ShutdownMessage{}
}
