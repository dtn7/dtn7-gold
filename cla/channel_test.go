package cla

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestMerge(t *testing.T) {
	const (
		packages0 = 1000
		packages1 = 4000
	)

	bndl, err := bundle.Builder().
		Source("dtn:src").
		Destination("dtn:dest").
		CreationTimestampEpoch().
		Lifetime("60s").
		BundleCtrlFlags(bundle.MustNotFragmented).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Error(err)
	}

	recBndl := NewConvergenceStatus(nil, bundle.DtnNone(), ReceivedBundle, &bndl)

	ch0 := make(chan ConvergenceStatus)
	ch1 := make(chan ConvergenceStatus)

	chMerge := merge(ch0, ch1)
	errCh := make(chan error, packages0+packages1)

	var counter sync.Map
	counter.Store("counter", packages0+packages1)

	go func() {
		for {
			select {
			case cs, ok := <-chMerge:
				if ok {
					if cs.MessageType != ReceivedBundle {
						errCh <- fmt.Errorf("Wrong MessageType %v", cs.MessageType)
					} else {
						c, _ := counter.Load("counter")
						cVal := c.(int) - 1
						counter.Store("counter", cVal)

						if recBndl := cs.Message.(*bundle.Bundle); !reflect.DeepEqual(recBndl, &bndl) {
							errCh <- fmt.Errorf("Received bundle differs: %v, %v", recBndl, &bndl)
						} else {
							errCh <- nil
						}

						if cVal == 0 {
							return
						}
					}
				}
			}
		}
	}()

	spam := func(ch chan ConvergenceStatus, amount int) {
		for i := 0; i < amount; i++ {
			ch <- recBndl
		}
		close(ch)
	}

	go spam(ch0, packages0)
	go spam(ch1, packages1)

	for i := 0; i < packages0+packages1; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	c, _ := counter.Load("counter")
	if c.(int) != 0 {
		t.Fatalf("Counter is not zero: %d", c.(int))
	}
}

func TestJoinReceivers(t *testing.T) {
	const (
		clients  = 50
		packages = 250
	)

	bndl, err := bundle.Builder().
		Source("dtn:src").
		Destination("dtn:dest").
		CreationTimestampEpoch().
		Lifetime("60s").
		BundleCtrlFlags(bundle.MustNotFragmented).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Error(err)
	}

	recBndl := NewConvergenceStatus(nil, bundle.DtnNone(), ReceivedBundle, &bndl)

	chns := make([]chan ConvergenceStatus, clients)
	for i := 0; i < clients; i++ {
		chns[i] = make(chan ConvergenceStatus)
	}

	chMerge := JoinReceivers(chns...)
	errCh := make(chan error, clients*packages)

	var counter sync.Map
	counter.Store("counter", clients*packages)

	go func() {
		for {
			select {
			case cs, ok := <-chMerge:
				if ok {
					if cs.MessageType != ReceivedBundle {
						errCh <- fmt.Errorf("Wrong MessageType %v", cs.MessageType)
					} else {
						c, _ := counter.Load("counter")
						cVal := c.(int) - 1
						counter.Store("counter", cVal)

						if recBndl := cs.Message.(*bundle.Bundle); !reflect.DeepEqual(recBndl, &bndl) {
							errCh <- fmt.Errorf("Received bundle differs: %v, %v", recBndl, &bndl)
						} else {
							errCh <- nil
						}

						if cVal == 0 {
							return
						}
					}
				}
			}
		}
	}()

	for i := 0; i < clients; i++ {
		go func(ch chan ConvergenceStatus) {
			for i := 0; i < packages; i++ {
				ch <- recBndl
			}
			close(ch)
		}(chns[i])
	}

	for i := 0; i < clients*packages; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	c, _ := counter.Load("counter")
	if c.(int) != 0 {
		t.Fatalf("Counter is not zero: %d", c.(int))
	}
}
