// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"github.com/dtn7/cboring"
)

func (s *Session) handleOut() {
	defer func() {
		// TODO
	}()

	for message := range s.outChannel {
		if err := cboring.Marshal(&message, s.Out); err != nil {
			s.logger().WithError(err).WithField("message", message).Error("Sending outgoing message errored")
			return
		}

		s.logger().WithField("message", message).Info("Sent outgoing message")
	}
}
