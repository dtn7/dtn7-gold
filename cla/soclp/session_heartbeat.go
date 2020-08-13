// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// heartbeatTicker is like a time.Ticker, but for altering intervals.
//
// Note: Go 1.15, which was released just some days after writing this, now supports this feature in the Ticker. Thus,
// this custom ticker can be removed once the minimum supported Go version is greater/equal 1.15.
type heartbeatTicker struct {
	C chan time.Time

	stopped     bool
	stoppedLock sync.Mutex
}

// newHeartbeatTicker which is scheduled for the given delay.
func newHeartbeatTicker(delay time.Duration) (ht *heartbeatTicker) {
	ht = &heartbeatTicker{C: make(chan time.Time)}
	ht.reschedule(delay)

	return
}

// reschedule a "wakey wakey" call for this ticker's channel.
func (ht *heartbeatTicker) reschedule(delay time.Duration) {
	go func() {
		defer func() { _ = recover() }()

		time.Sleep(delay)

		ht.stoppedLock.Lock()
		defer ht.stoppedLock.Unlock()

		if ht.stopped {
			close(ht.C)
		} else {
			ht.C <- time.Now()
		}
	}()
}

// stop this ticker.
func (ht *heartbeatTicker) stop() {
	ht.stoppedLock.Lock()
	defer ht.stoppedLock.Unlock()

	ht.stopped = true
}

func (s *Session) handleHeartbeat() {
	receiveCheck := newHeartbeatTicker(s.HeartbeatTimeout)
	sentCheck := newHeartbeatTicker(s.HeartbeatTimeout / 2)

	defer receiveCheck.stop()
	defer sentCheck.stop()

	defer s.closeAction()

	for {
		select {
		case <-s.heartbeatStopChannel:
			return

		case now := <-receiveCheck.C:
			if s.checkHeartbeatReceive(receiveCheck, now) {
				return
			}

		case now := <-sentCheck.C:
			s.checkHeartbeatSent(sentCheck, now)
		}
	}
}

// checkHeartbeatReceive compares the last received message's time to check if the timeout has expired.
func (s *Session) checkHeartbeatReceive(receiveCheck *heartbeatTicker, now time.Time) (expired bool) {
	s.lastReceiveLock.RLock()
	defer s.lastReceiveLock.RUnlock()

	delta := s.lastReceive.Add(s.HeartbeatTimeout).Sub(now)

	s.logger().WithFields(log.Fields{
		"last-receive": s.lastReceive,
		"now":          now,
		"delta":        delta,
		"timeout":      s.HeartbeatTimeout,
	}).Debug("Heartbeat check for incoming messages")

	if delta < 0 {
		s.logger().WithFields(log.Fields{
			"last-receive": s.lastReceive,
			"timeout":      s.HeartbeatTimeout,
		}).Warn("Received no incoming messages within timeout")

		return true
	} else {
		receiveCheck.reschedule(delta / 2)

		return false
	}
}

// checkHeartbeatSent inspects the time before hitting the timeout for outgoing messages.
//
// If the time delta between now and the deadline is smaller or equal 1/8 of the heartbeat timeout, a heartbeat status
// message will be sent. Otherwise, another check is rescheduled in the half of the delta's time.
func (s *Session) checkHeartbeatSent(sentCheck *heartbeatTicker, now time.Time) {
	s.lastSentLock.RLock()
	defer s.lastSentLock.RUnlock()

	delta := s.lastSent.Add(s.HeartbeatTimeout).Sub(now)

	s.logger().WithFields(log.Fields{
		"last-sent": s.lastSent,
		"now":       now,
		"delta":     delta,
		"timeout":   s.HeartbeatTimeout,
	}).Debug("Heartbeat check for outgoing messages")

	if delta <= s.HeartbeatTimeout/8 {
		s.outChannel <- Message{MessageType: NewHeartbeatStatusMessage()}
		s.logger().Debug("Sent heartbeat status message")

		sentCheck.reschedule(s.HeartbeatTimeout / 2)
	} else {
		sentCheck.reschedule(delta / 2)
	}
}
