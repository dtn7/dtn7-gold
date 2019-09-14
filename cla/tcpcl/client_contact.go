package tcpcl

import (
	log "github.com/sirupsen/logrus"
)

// This file contains code for the Client's contact state.

// handleContact manges the contact state for the Contact Header exchange.
func (client *TCPCLClient) handleContact() error {
	var logger = log.WithFields(log.Fields{
		"session": client,
		"state":   "contact",
	})

	switch {
	case client.active && !client.contactSent, !client.active && !client.contactSent && client.contactRecv:
		client.chSent = NewContactHeader(0)
		if err := client.chSent.Marshal(client.rw); err != nil {
			return err
		} else if err := client.rw.Flush(); err != nil {
			return err
		} else {
			client.contactSent = true
			logger.WithField("msg", client.chSent).Debug("Sent Contact Header")
		}

	case !client.active && !client.contactRecv, client.active && client.contactSent && !client.contactRecv:
		if err := client.chRecv.Unmarshal(client.rw); err != nil {
			return err
		} else {
			client.contactRecv = true
			logger.WithField("msg", client.chRecv).Debug("Received Contact Header")
		}

	case client.contactSent && client.contactRecv:
		// TODO: check contact header flags
		logger.Debug("Exchanged Contact Headers")
		client.state.Next()
	}

	return nil
}
