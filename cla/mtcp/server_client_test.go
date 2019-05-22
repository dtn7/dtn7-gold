package mtcp

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dtn7/dtn7/bundle"
)

func getRandomPort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Error(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

func TestMTCPServerClient(t *testing.T) {
	// Address
	port := getRandomPort(t)

	// Bundle
	const (
		clients  = 100
		packages = 1000
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
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(clients + 1) // 1 for the server

	// Server
	serv := NewMTCPServer(
		fmt.Sprintf(":%d", port), bundle.MustNewEndpointID("dtn:mtcpcla"), false)
	if err, _ := serv.Start(); err != nil {
		t.Fatal(err)
	}

	var counter sync.Map
	counter.Store("counter", clients*packages)

	go func() {
		var chnl = serv.Channel()

		for {
			select {
			case b := <-chnl:
				c, _ := counter.Load("counter")
				cVal := c.(int) - 1
				counter.Store("counter", cVal)

				if !reflect.DeepEqual(*b.Bundle, bndl) {
					t.Errorf("Received bundle differs: %v, %v", b, bndl)
				}

				if cVal == 0 {
					serv.Close()
					wg.Done()
					return
				}

			case <-time.After(time.Second):
				t.Fatal("Server timed out")
			}
		}
	}()

	// Client
	for c := 0; c < clients; c++ {
		go func() {
			client := NewAnonymousMTCPClient(fmt.Sprintf("localhost:%d", port), false)
			if err, _ := client.Start(); err != nil {
				t.Fatal(err)
			}

			for i := 0; i < packages; i++ {
				if err := client.Send(&bndl); err != nil {
					t.Fatal(err)
				}
			}

			client.Close()
			wg.Done()
		}()
	}

	wg.Wait()

	c, _ := counter.Load("counter")
	if c.(int) != 0 {
		t.Fatalf("Counter is not zero: %d", c.(int))
	}
}
