// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"time"

	"github.com/dtn7/cboring"
)

func (s *Session) handleOut() {
	defer s.closeAction()

	for {
		select {
		case message := <-s.outChannel:
			if err := cboring.Marshal(&message, s.Out); err != nil {
				s.logger().WithError(err).WithField("message", message).Error("Sending outgoing message errored")
				return
			}

			s.logger().WithField("message", message).Info("Sent outgoing message")

			s.updateLastSent()

		case <-s.outStopChannel:
			return
		}
	}
}

// updateLastSent sets lastSent to the current time.
func (s *Session) updateLastSent() {
	s.lastSentLock.Lock()
	defer s.lastSentLock.Unlock()

	s.lastSent = time.Now()
	s.logger().WithField("last-sent", s.lastSent).Debug("Updated last sent timestamp")
}
