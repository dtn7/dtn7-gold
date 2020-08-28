// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"testing"
	"time"
)

func TestKeepaliveTicker(t *testing.T) {
	ticker := NewKeepaliveTicker()

	intervals := []time.Duration{50 * time.Millisecond, 75 * time.Millisecond, 100 * time.Millisecond}
	for _, interval := range intervals {
		ticker.Reschedule(interval)

		// wait for tick
		select {
		case <-ticker.C:
		case <-time.After(2 * interval):
			t.Fatalf("timeout at %v", interval)
		}

		// no second tick should occur
		select {
		case <-ticker.C:
			t.Fatalf("second tick at %v", interval)
		case <-time.After(2 * interval):
		}
	}

	ticker.Reschedule(50 * time.Millisecond)
	ticker.Stop()

	select {
	case <-ticker.C:
		t.Fatal("no tick was expected after Stop")
	case <-time.After(100 * time.Millisecond):
	}
}
