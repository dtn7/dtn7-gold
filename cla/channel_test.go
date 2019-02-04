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

	ch0 := make(chan bundle.Bundle)
	ch1 := make(chan bundle.Bundle)

	chMerge := merge(ch0, ch1)

	go func() {
		var counter int = packages0 + packages1

		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					counter--
					if !reflect.DeepEqual(b, bndl) {
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

	spam := func(ch chan bundle.Bundle, amount int) {
		for i := 0; i < amount; i++ {
			ch <- bndl
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

	chns := make([]chan bundle.Bundle, clients)
	for i := 0; i < clients; i++ {
		chns[i] = make(chan bundle.Bundle)
	}

	chMerge := JoinReceivers(chns...)

	go func() {
		var counter int = clients * packages

		for {
			select {
			case b, ok := <-chMerge:
				if ok {
					counter--
					if !reflect.DeepEqual(b, bndl) {
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
		go func(ch chan bundle.Bundle) {
			for i := 0; i < packages; i++ {
				ch <- bndl
			}
			close(ch)
		}(chns[i])
	}

	time.Sleep(100 * time.Millisecond)
}
