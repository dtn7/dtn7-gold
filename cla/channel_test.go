package cla

import (
	"reflect"
	"testing"
	"time"

	"github.com/geistesk/dtn7/bundle"
)

func TestMerge(t *testing.T) {
	const (
		packages0 = 1000
		packages1 = 4000
	)

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn", "dest"),
			bundle.MustNewEndpointID("dtn", "src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Error(err)
	}

	recBndl := NewRecBundle(bndl, bundle.DtnNone())

	ch0 := make(chan RecBundle)
	ch1 := make(chan RecBundle)

	chMerge := merge(ch0, ch1)

	go func() {
		var counter int = packages0 + packages1

		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					counter--
					if !reflect.DeepEqual(b.Bundle, bndl) {
						t.Errorf("Received bundle differs: %v, %v", b, bndl)
					}
				}

			case <-time.After(time.Millisecond):
				if counter != 0 {
					t.Errorf("Counter is not zero: %d", counter)
				}

				close(chMerge)
				return
			}
		}
	}()

	spam := func(ch chan RecBundle, amount int) {
		for i := 0; i < amount; i++ {
			ch <- recBndl
		}
		close(ch)
	}

	go spam(ch0, packages0)
	go spam(ch1, packages1)

	time.Sleep(10 * time.Millisecond)
}

func TestJoinReceivers(t *testing.T) {
	const (
		clients  = 1000
		packages = 10000
	)

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn", "dest"),
			bundle.MustNewEndpointID("dtn", "src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Error(err)
	}

	recBndl := NewRecBundle(bndl, bundle.DtnNone())

	chns := make([]chan RecBundle, clients)
	for i := 0; i < clients; i++ {
		chns[i] = make(chan RecBundle)
	}

	chMerge := JoinReceivers(chns...)

	go func() {
		var counter int = clients * packages

		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					counter--
					if !reflect.DeepEqual(b.Bundle, bndl) {
						t.Errorf("Received bundle differs: %v, %v", b, bndl)
					}
				}

			case <-time.After(time.Millisecond):
				if counter != 0 {
					t.Errorf("Counter is not zero: %d", counter)
				}

				close(chMerge)
				return
			}
		}
	}()

	for i := 0; i < clients; i++ {
		go func(ch chan RecBundle) {
			for i := 0; i < packages; i++ {
				ch <- recBndl
			}
			close(ch)
		}(chns[i])
	}

	time.Sleep(10 * time.Millisecond)
}
