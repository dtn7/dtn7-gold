// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
)

// This file contains code for the Client's contact state.

// handleContact manges the contact state for the Contact Header exchange.
func (client *Client) handleContact() error {
	switch {
	case client.active && !client.contactSent, !client.active && !client.contactSent && client.contactRecv:
		client.chSent = NewContactHeader(0)
		client.contactSent = true

		client.msgsOut <- &client.chSent
		client.log().WithField("msg", client.chSent).Debug("Sent Contact Header")

	case !client.active && !client.contactRecv, client.active && client.contactSent && !client.contactRecv:
		msg := <-client.msgsIn
		switch msg := msg.(type) {
		case *ContactHeader:
			client.chRecv = *msg
			client.contactRecv = true
			client.log().WithField("msg", client.chRecv).Debug("Received Contact Header")

		default:
			client.log().WithField("msg", msg).Warn("Received wrong message")
			return fmt.Errorf("Wrong message type")
		}

	case client.contactSent && client.contactRecv:
		// TODO: check contact header flags
		client.log().Debug("Exchanged Contact Headers")
		client.state.Next()
	}

	return nil
}
