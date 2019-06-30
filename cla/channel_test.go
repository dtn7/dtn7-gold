package cla

import (
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

	recBndl := NewRecBundle(&bndl, bundle.DtnNone())

	ch0 := make(chan RecBundle)
	ch1 := make(chan RecBundle)

	chMerge := merge(ch0, ch1)

	var wg sync.WaitGroup
	wg.Add(2 + 1) // 2 clients, 1 server

	var counter sync.Map
	counter.Store("counter", packages0+packages1)

	go func() {
		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					c, _ := counter.Load("counter")
					cVal := c.(int) - 1
					counter.Store("counter", cVal)

					if !reflect.DeepEqual(*b.Bundle, bndl) {
						t.Errorf("Received bundle differs: %v, %v", b, bndl)
					}

					if cVal == 0 {
						wg.Done()
						return
					}
				}
			}
		}
	}()

	spam := func(ch chan RecBundle, amount int) {
		for i := 0; i < amount; i++ {
			ch <- recBndl
		}
		close(ch)

		wg.Done()
	}

	go spam(ch0, packages0)
	go spam(ch1, packages1)

	wg.Wait()

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

	recBndl := NewRecBundle(&bndl, bundle.DtnNone())

	chns := make([]chan RecBundle, clients)
	for i := 0; i < clients; i++ {
		chns[i] = make(chan RecBundle)
	}

	chMerge := JoinReceivers(chns...)

	var wg sync.WaitGroup
	wg.Add(clients + 1) // 1 for the server

	var counter sync.Map
	counter.Store("counter", clients*packages)

	go func() {
		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					c, _ := counter.Load("counter")
					cVal := c.(int) - 1
					counter.Store("counter", cVal)

					if !reflect.DeepEqual(*b.Bundle, bndl) {
						t.Errorf("Received bundle differs: %v, %v", b, bndl)
					}

					if cVal == 0 {
						wg.Done()
						return
					}
				}
			}
		}
	}()

	for i := 0; i < clients; i++ {
		go func(ch chan RecBundle) {
			for i := 0; i < packages; i++ {
				ch <- recBndl
			}
			close(ch)

			wg.Done()
		}(chns[i])
	}

	wg.Wait()

	c, _ := counter.Load("counter")
	if c.(int) != 0 {
		t.Fatalf("Counter is not zero: %d", c.(int))
	}
}
