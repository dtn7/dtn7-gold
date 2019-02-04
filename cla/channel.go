package cla

import (
	"sync"
)

// merge merges two RecBundle channels into a new one.
func merge(a, b <-chan RecBundle) (ch chan RecBundle) {
	var wg sync.WaitGroup
	wg.Add(2)

	ch = make(chan RecBundle)

	for _, c := range []<-chan RecBundle{a, b} {
		go func(c <-chan RecBundle) {
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

// JoinReceivers joins the given receiving bundle channels into a new channel,
// containing all bundles from all channels.
func JoinReceivers(chans ...chan RecBundle) chan RecBundle {
	switch len(chans) {
	case 0:
		ch := make(chan RecBundle)
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
