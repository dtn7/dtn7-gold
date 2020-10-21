// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package mtcp

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
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

	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port
}

func TestMTCPServerClient(t *testing.T) {
	// Address
	port := getRandomPort(t)

	// Bundle
	const (
		clients  = 25
		packages = 100
	)

	bndl, err := bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("60s").
		BundleCtrlFlags(bpv7.MustNotFragmented).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	// Server
	serv := NewMTCPServer(
		fmt.Sprintf(":%d", port), bpv7.MustNewEndpointID("dtn://mtcpcla/"), false)
	if err, _ := serv.Start(); err != nil {
		t.Fatal(err)
	}

	var counter sync.Map
	counter.Store("counter", clients*packages)

	errCh := make(chan error, clients*packages*2)

	go func() {
		for cs := range serv.Channel() {
			if cs.MessageType != cla.ReceivedBundle {
				errCh <- fmt.Errorf("Wrong MessageType %v", cs.MessageType)
			} else {
				c, _ := counter.Load("counter")
				cVal := c.(int) - 1
				counter.Store("counter", cVal)

				recBndl := cs.Message.(cla.ConvergenceReceivedBundle).Bundle
				if !reflect.DeepEqual(recBndl, &bndl) {
					errCh <- fmt.Errorf("Received bundle differs: %v, %v", recBndl, &bndl)
				} else {
					errCh <- nil
				}

				if cVal == 0 {
					if err := serv.Close(); err != nil {
						errCh <- err
					}
					return
				}
			}
		}
	}()

	// Clients
	for c := 0; c < clients; c++ {
		go func() {
			client := NewAnonymousMTCPClient(fmt.Sprintf("localhost:%d", port), false)
			if err, _ := client.Start(); err != nil {
				errCh <- fmt.Errorf("Starting Client failed: %v", err)
				return
			}

			// Dry each client's channel
			go func(client cla.ConvergenceSender) {
				for range client.Channel() {
				}
			}(client)

			for i := 0; i < packages; i++ {
				errCh <- client.Send(bndl)
			}

			if err := client.Close(); err != nil {
				errCh <- err
				return
			}
		}()
	}

	for i := 0; i < clients*packages*2; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	c, _ := counter.Load("counter")
	if c.(int) != 0 {
		t.Fatalf("Counter is not zero: %d", c.(int))
	}
}
