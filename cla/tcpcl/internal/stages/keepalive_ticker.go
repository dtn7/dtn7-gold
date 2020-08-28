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
	// c is the internal channel. External calls should use C, which is the same just with directions.
	c chan time.Time

	// C sends ticks with the current time.
	C <-chan time.Time

	// stopped is set to != 0 if this ticker was stopped
	stopped uint32
}

// newKeepaliveTicker which needs to be scheduled by calling Reschedule.
func newKeepaliveTicker() *keepaliveTicker {
	c := make(chan time.Time)
	return &keepaliveTicker{
		c:       c,
		C:       c,
		stopped: 0,
	}
}

// Reschedule a tick for this ticker's channel C.
func (ticker *keepaliveTicker) Reschedule(delay time.Duration) {
	if atomic.LoadUint32(&ticker.stopped) != 0 {
		return
	}

	go func() {
		time.Sleep(delay)

		if atomic.LoadUint32(&ticker.stopped) == 0 {
			ticker.c <- time.Now()
		}
	}()
}

// Stop this ticker.
//
// The internal channel C will NOT be closed to prevent reading the closing as an erroneous tick.
func (ticker *keepaliveTicker) Stop() {
	atomic.StoreUint32(&ticker.stopped, 1)
}
