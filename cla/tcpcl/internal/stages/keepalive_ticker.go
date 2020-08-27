// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"sync/atomic"
	"time"
)

// keepaliveTicker is a variant of the time.Ticker which works like a wind-up clock.
//
// The next tick of its channel C can be programmed by calling Reschedule. Multiple ticks might be scheduled.
// The internal channel C will NOT be closed to prevent reading the closing as an erroneous tick.
type keepaliveTicker struct {
	C chan time.Time

	// stopped is set to != 0 if this ticker was stopped
	stopped uint32
}

// newKeepaliveTicker which is scheduled for the given delay.
func newKeepaliveTicker(delay time.Duration) (ticker *keepaliveTicker) {
	ticker = &keepaliveTicker{C: make(chan time.Time)}
	ticker.Reschedule(delay)

	return
}

// Reschedule a "wakey wakey" call for this ticker's channel.
func (ticker *keepaliveTicker) Reschedule(delay time.Duration) {
	if atomic.LoadUint32(&ticker.stopped) != 0 {
		return
	}

	go func() {
		time.Sleep(delay)

		if atomic.LoadUint32(&ticker.stopped) == 0 {
			ticker.C <- time.Now()
		}
	}()
}

// Stop this ticker.
//
// The internal channel C will NOT be closed to prevent reading the closing as an erroneous tick.
func (ticker *keepaliveTicker) Stop() {
	atomic.StoreUint32(&ticker.stopped, 1)
}
