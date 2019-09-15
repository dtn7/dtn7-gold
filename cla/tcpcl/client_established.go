package tcpcl

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
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
			var diff = time.Since(client.keepaliveLast)
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

	case XFER_SEGMENT:
		var dataTransMsg DataTransmissionMessage
		if err := dataTransMsg.Unmarshal(client.rw); err != nil {
			return err
		} else {
			logger.WithField("msg", dataTransMsg).Info("Received XFER_SEGMENT")
		}

	default:
		logger.WithField("magic", nextMsg).Debug("Received unsupported magic")
	}

	return nil
}

func (client *TCPCLClient) Send(bndl *bundle.Bundle) error {
	var logger = log.WithField("session", client)
	var t = NewBundleTransfer(23, *bndl)

	for {
		dtm, err := t.NextSegment(client.segmentMru)

		if err == io.EOF {
			logger.Info("Finished Transfer")
			return nil
		} else if err != nil {
			logger.WithError(err).Warn("Fetching Segment errored")
			return err
		}

		if err := dtm.Marshal(client.rw); err != nil {
			logger.WithField("msg", dtm).WithError(err).Warn(
				"Sending XFER_SEGMENT errored")
			return err
		} else {
			logger.WithField("msg", dtm).Debug("Sent XFER_SEGMENT")
		}
	}
}
