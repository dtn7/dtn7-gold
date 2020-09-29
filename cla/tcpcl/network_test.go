// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func testGetRandomData(size int) []byte {
	payload := make([]byte, size)

	rand.Seed(0)
	rand.Read(payload)

	return payload
}

func getRandomPort(t *testing.T) (port int) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	port = l.Addr().(*net.TCPAddr).Port

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	return
}

func handleListener(serverAddr string, msgs, clients int, clientWg, serverWg *sync.WaitGroup, errs chan error) {
	var (
		msgsRecv  uint32
		msgsApprd uint32
	)

	defer serverWg.Done()

	manager := cla.NewManager()
	manager.Register(NewListener(serverAddr, bundle.MustNewEndpointID("dtn://server/")))

	go func() {
		for {
			switch cs := <-manager.Channel(); cs.MessageType {
			case cla.ReceivedBundle:
				atomic.AddUint32(&msgsRecv, 1)

			case cla.PeerAppeared:
				atomic.AddUint32(&msgsApprd, 1)

				/*
					if sender, ok := cs.Sender.(cla.ConvergenceSender); !ok {
						errs <- fmt.Errorf("listener: new peer is not a ConvergenceSender; %v", cs)
					} else {

						bndl, err := bundle.Builder().
							CRC(bundle.CRC32).
							Source("dtn://server/").
							Destination(cs.Message).
							CreationTimestampNow().
							Lifetime(30 * time.Minute).
							HopCountBlock(64).
							PayloadBlock([]byte("hello back!")).
							Build()
						if err != nil {
							errs <- fmt.Errorf("listener: %w", err)
						} else if err = sender.Send(&bndl); err != nil {
							errs <- fmt.Errorf("listener: %w", err)
						}
					}
				*/
			}
		}
	}()

	clientWg.Wait()
	// Wait for last transmission to be finished
	time.Sleep(time.Second)

	logrus.Info("Closing listener / manager")

	manager.Close()

	if r := atomic.LoadUint32(&msgsRecv); r != uint32(msgs*clients) {
		errs <- fmt.Errorf("listener received %d messages instead of %d", r, msgs*clients)
	}
	if a := atomic.LoadUint32(&msgsApprd); a != uint32(clients) {
		errs <- fmt.Errorf("listener received %d appeared peers instead of %d", a, clients)
	}
}

func handleClient(serverAddr string, clientNo, msgs, payload int, clientWg *sync.WaitGroup, errs chan error) {
	defer clientWg.Done()

	clientEid := fmt.Sprintf("dtn://client-%d/", clientNo)
	client := DialClient(serverAddr, bundle.MustNewEndpointID(clientEid), false)
	if err, _ := client.Start(); err != nil {
		errs <- fmt.Errorf("client %d: %w", clientNo, err)
		return
	}

	time.Sleep(time.Second)

	var thisClientWg sync.WaitGroup
	thisClientWg.Add(1)

	go func() {
		for {
			switch cs := <-client.Channel(); cs.MessageType {
			case cla.ReceivedBundle:
				// thisClientWg.Done()
			}
		}
	}()

	go func() {
		defer thisClientWg.Done()

		for i := 0; i < msgs; i++ {
			bndl, err := bundle.Builder().
				CRC(bundle.CRC32).
				Source(clientEid).
				Destination("dtn://server/").
				CreationTimestampNow().
				Lifetime(30 * time.Minute).
				HopCountBlock(64).
				PayloadBlock(testGetRandomData(payload)).
				Build()

			if err != nil {
				errs <- fmt.Errorf("client %d: %w", clientNo, err)
				return
			} else if err := client.Send(&bndl); err != nil {
				errs <- fmt.Errorf("client %d: %w", clientNo, err)
				return
			}
		}
	}()

	thisClientWg.Wait()
	time.Sleep(time.Second)

	logrus.WithField("client", clientNo).Info("Closing client")

	client.Close()
}

func startTestTCPCLNetwork(msgs, clients, payload int, t *testing.T) {
	var serverAddr = fmt.Sprintf("localhost:%d", getRandomPort(t))
	var errs = make(chan error)

	var clientWg sync.WaitGroup
	var serverWg sync.WaitGroup

	clientWg.Add(clients)
	serverWg.Add(1)

	go handleListener(serverAddr, msgs, clients, &clientWg, &serverWg, errs)
	time.Sleep(250 * time.Millisecond)

	for i := 0; i < clients; i++ {
		go handleClient(serverAddr, i, msgs, payload, &clientWg, errs)
	}

	go func() {
		serverWg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTCPCLNetwork(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	tests := []struct {
		clients int
		msgs    int
		payload int
	}{
		{1, 1, 64},
		{1, 1, 1048576},
		{1, 256, 1024},
		{2, 1, 64},
		{2, 1, 1048576},
		{2, 256, 1024},
		{64, 1, 1024},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			startTestTCPCLNetwork(test.msgs, test.clients, test.payload, t)
		})
	}
}
