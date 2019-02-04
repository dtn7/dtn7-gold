package cla

import (
	"sync"

	"github.com/geistesk/dtn7/bundle"
)

// merge merges two bundle channels into a new one.
func merge(a, b <-chan bundle.Bundle) (ch chan bundle.Bundle) {
	var wg sync.WaitGroup
	wg.Add(2)

	ch = make(chan bundle.Bundle)

	for _, c := range []<-chan bundle.Bundle{a, b} {
		go func(c <-chan bundle.Bundle) {
			for bndl := range c {
				ch <- bndl
			}

			wg.Done()
		}(c)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return
}

// JoinReceivers joins the given bundle channels into a new channel, receiving
// all bundles from all channels.
func JoinReceivers(chans ...chan bundle.Bundle) chan bundle.Bundle {
	switch len(chans) {
	case 0:
		ch := make(chan bundle.Bundle)
		close(ch)
		return ch

	case 1:
		return chans[0]

	default:
		pivot := len(chans) / 2

		left := JoinReceivers(chans[pivot:]...)
		right := JoinReceivers(chans[:pivot]...)

		return merge(left, right)
	}
}
