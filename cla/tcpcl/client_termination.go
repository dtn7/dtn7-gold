package tcpcl

import (
	log "github.com/sirupsen/logrus"
)

// This file contains code for the Client's termination state.

// terminate sends a SESS_TERM message to its peer and closes the session afterwards.
func (client *TCPCLClient) terminate(code SessionTerminationCode) {
	var logger = log.WithField("session", client)

	var sessTerm = NewSessionTerminationMessage(0, code)
	if err := sessTerm.Marshal(client.rw); err != nil {
		logger.WithError(err).Warn("Failed to send session termination message")
	} else if err := client.rw.Flush(); err != nil {
		logger.WithError(err).Warn("Failed to flush buffer")
	} else if err := client.conn.Close(); err != nil {
		logger.WithError(err).Warn("Failed to close TCP connection")
	} else {
		logger.Info("Terminated session")
	}
}
