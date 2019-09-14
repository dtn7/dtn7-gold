package tcpcl

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// This file contains code for the Client's established state.

// keepaliveHandler handles the KEEPALIVE messages.
func (client *TCPCLClient) keepaliveHandler() {
	var logger = log.WithField("session", client)

	var keepaliveTicker = time.NewTicker(time.Duration(client.keepalive) * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-keepaliveTicker.C:
			// Send a keepalive
			var keepaliveMsg = NewKeepaliveMessage()
			if err := keepaliveMsg.Marshal(client.rw); err != nil {
				logger.WithError(err).Warn("Sending KEEPALIVE errored")
			} else if err := client.rw.Flush(); err != nil {
				logger.WithError(err).Warn("Flushing KEEPALIVE errored")
			} else {
				logger.WithField("msg", keepaliveMsg).Debug("Sent KEEPALIVE message")
			}
			// TODO: terminate session

			// Check last received keepalive
			var diff = time.Now().Sub(client.keepaliveLast)
			if diff > 2*time.Duration(client.keepalive)*time.Second {
				logger.WithFields(log.Fields{
					"last keepalive": client.keepaliveLast,
					"interval":       time.Duration(client.keepalive) * time.Second,
				}).Warn("No KEEPALIVE was received within expected time frame")

				// TODO: terminate session
			} else {
				logger.WithField("last keepalive", client.keepaliveLast).Debug(
					"Received last KEEPALIVE within expected time frame")
			}

		case <-client.keepaliveStopSyn:
			close(client.keepaliveStopAck)
			return
		}
	}
}

// handleEstablished manges the established state.
func (client *TCPCLClient) handleEstablished() error {
	var logger = log.WithField("session", client)

	if !client.keepaliveStarted {
		go client.keepaliveHandler()
		client.keepaliveStarted = true
	}

	nextMsg, nextMsgErr := client.rw.ReadByte()
	if nextMsgErr != nil {
		return nextMsgErr
	} else if err := client.rw.UnreadByte(); err != nil {
		return err
	}

	switch nextMsg {
	case KEEPALIVE:
		var keepaliveMsg KeepaliveMessage
		if err := keepaliveMsg.Unmarshal(client.rw); err != nil {
			return err
		} else {
			client.keepaliveLast = time.Now()
			logger.WithField("msg", keepaliveMsg).Debug("Received KEEPALIVE message")
		}

	case SESS_TERM:
		var sesstermMsg SessionTerminationMessage
		if err := sesstermMsg.Unmarshal(client.rw); err != nil {
			return err
		} else {
			logger.WithField("msg", sesstermMsg).Info("Received SESS_TERM")
			client.state.Terminate()
		}

	default:
		logger.WithField("magic", nextMsg).Debug("Received unsupported magic")
	}

	return nil
}
