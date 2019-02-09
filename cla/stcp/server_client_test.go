package stcp

import (
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/geistesk/dtn7/bundle"
)

func getRandomPort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Error(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

func TestSTCPServerClient(t *testing.T) {
	// Address
	port := getRandomPort(t)

	// Bundle
	const (
		packages = 1000
		clients  = 100
	)

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Error(err)
	}

	// Server
	serv := NewSTCPServer(
		fmt.Sprintf(":%d", port), bundle.MustNewEndpointID("dtn:stcpcla"))

	go func() {
		var counter int = packages * clients
		var chnl = serv.Channel()

		for {
			select {
			case b := <-chnl:
				counter--
				if !reflect.DeepEqual(b.Bundle, bndl) {
					t.Errorf("Received bundle differs: %v, %v", b, bndl)
				}

			case <-time.After(time.Millisecond):
				serv.Close()
				if counter != 0 {
					t.Fatalf("Counter is not zero: %d", counter)
				}
				break
			}
		}
	}()

	// Client
	for c := 0; c < clients; c++ {
		go func() {
			client, err := NewAnonymousSTCPClient(fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Error(err)
			}

			for i := 0; i < packages; i++ {
				if err = client.Send(bndl); err != nil {
					t.Error(err)
				}
			}

			client.Close()
		}()
	}

	time.Sleep(10 * time.Millisecond)
}
