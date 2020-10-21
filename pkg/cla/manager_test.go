// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cla

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func TestManager(t *testing.T) {
	const (
		senderNo   int = 25
		receiverNo int = 200
	)

	var bndl, bndlErr = bpv7.Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("10m").
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world")).
		Build()
	if bndlErr != nil {
		t.Fatal(bndlErr)
	}

	/* Setup */
	var manager = NewManager()
	defer func() { _ = manager.Close() }()

	// Read the Manager's outbounding channel
	var readErrCh = make(chan error, receiverNo)
	go func(ch chan ConvergenceStatus) {
		for cs := range ch {
			switch cs.MessageType {
			case ReceivedBundle:
				crb := cs.Message.(ConvergenceReceivedBundle)
				if !reflect.DeepEqual(crb.Bundle, &bndl) {
					readErrCh <- fmt.Errorf("Received bundle did not match")
				} else {
					readErrCh <- nil
				}

			default:
				// We don't care about other ConvergenceStatus types.
				// Those were already inspected by the Manager and have no value for us.
			}
		}
	}(manager.Channel())

	var sender [senderNo]ConvergenceSender
	for i := 0; i < senderNo; i++ {
		sender[i] = newMockConvSender(
			true, fmt.Sprintf("mock://sender_%d/", i),
			bpv7.MustNewEndpointID(fmt.Sprintf("dtn://ms_%d/", i)))

		manager.Register(sender[i])
	}

	var receiver [receiverNo]ConvergenceReceiver
	for i := 0; i < receiverNo; i++ {
		receiver[i] = newMockConvRec(
			true, fmt.Sprintf("mock://receiver_%d/", i),
			bpv7.MustNewEndpointID(fmt.Sprintf("dtn://mr_%d/", i)))

		manager.Register(receiver[i])
	}

	if css := manager.Sender(); len(css) != senderNo {
		t.Fatalf("Wrong amount of senders, expected: %d, got: %d", senderNo, len(css))
	}

	if crs := manager.Receiver(); len(crs) != receiverNo {
		t.Fatalf("Wrong amount of receiver, expected: %d, got: %d", receiverNo, len(crs))
	}

	/* Receive some bundles */
	var recWg sync.WaitGroup
	recWg.Add(receiverNo)

	for i := 0; i < receiverNo; i++ {
		go func(m *mockConvRec) {
			m.reportChan <- NewConvergenceReceivedBundle(m, m.GetEndpointID(), &bndl)
			recWg.Done()
		}(receiver[i].(*mockConvRec))
	}

	recWg.Wait()

	// Give the Manager some time to process the requests
	time.Sleep(10 * time.Duration(receiverNo) * time.Millisecond)

	/* Indicating failing CLAs, those should be restarted by the Manager */
	for i := 0; i < senderNo; i++ {
		go func(m *mockConvSender, i int) {
			if i >= senderNo/2 {
				m.reportChan <- NewConvergencePeerDisappeared(m, m.GetPeerEndpointID())
			}
		}(sender[i].(*mockConvSender), i)
	}

	// Give the Manager some time to process the requests
	time.Sleep(10 * time.Duration(senderNo/2) * time.Millisecond)

	/* Check results */
	for i := 0; i < receiverNo; i++ {
		if err := <-readErrCh; err != nil {
			t.Fatal(err)
		}
	}
}
