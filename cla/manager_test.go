package cla

import (
	"fmt"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestManager(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	const (
		senderNo   int = 25
		receiverNo int = 200
	)

	var bndl, bndlErr = bundle.Builder().
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
	defer manager.Close()

	// Read the Manager's outbounding channel
	var readErrCh = make(chan error, receiverNo)
	go func(ch chan ConvergenceStatus) {
		for cs := range ch {
			switch cs.MessageType {
			case ReceivedBundle:
				readErrCh <- nil

			default:
				readErrCh <- fmt.Errorf("Unsupported MessageType %v", cs.MessageType)
			}
		}
	}(manager.Channel())

	var sender [senderNo]ConvergenceSender
	for i := 0; i < senderNo; i++ {
		sender[i] = newMockConvSender(
			true, fmt.Sprintf("mock://sender_%d/", i),
			bundle.MustNewEndpointID(fmt.Sprintf("dtn://ms_%d/", i)))

		if err := manager.Register(sender[i]); err != nil {
			t.Fatal(err)
		}
	}

	var receiver [receiverNo]ConvergenceReceiver
	for i := 0; i < receiverNo; i++ {
		receiver[i] = newMockConvRec(
			true, fmt.Sprintf("mock://receiver_%d/", i),
			bundle.MustNewEndpointID(fmt.Sprintf("dtn://mr_%d/", i)))

		if err := manager.Register(receiver[i]); err != nil {
			t.Fatal(err)
		}
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

	// Give the Manager some time to process the bundles
	time.Sleep(10 * time.Duration(receiverNo) * time.Millisecond)

	/* Check results */
	for i := 0; i < receiverNo; i++ {
		if err := <-readErrCh; err != nil {
			t.Fatal(err)
		}
	}
}
