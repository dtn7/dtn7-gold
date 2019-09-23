package tcpcl

import (
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// This file contains code for the Client's established state.

// handleEstablished manges the established state.
func (client *TCPCLClient) handleEstablished() (err error) {
	defer func() {
		if err != nil && client.keepaliveStarted {
			client.keepaliveTicker.Stop()
		}
	}()

	if !client.keepaliveStarted {
		client.keepaliveTicker = time.NewTicker(time.Duration(client.keepalive) * time.Second)
		client.keepaliveLast = time.Now()
		client.keepaliveStarted = true
	}

	select {
	case <-client.keepaliveTicker.C:
		// Send a keepalive
		var keepaliveMsg = NewKeepaliveMessage()
		client.msgsOut <- &keepaliveMsg
		client.log().WithField("msg", keepaliveMsg).Debug("Sent KEEPALIVE message")

		// Check last received keepalive
		var diff = time.Since(client.keepaliveLast)
		if diff > 2*time.Duration(client.keepalive)*time.Second {
			client.log().WithFields(log.Fields{
				"last keepalive": client.keepaliveLast,
				"interval":       time.Duration(client.keepalive) * time.Second,
			}).Warn("No KEEPALIVE was received within expected time frame")

			return fmt.Errorf("No KEEPALIVE was received within expected time frame")
		} else {
			client.log().WithField("last keepalive", client.keepaliveLast).Debug(
				"Received last KEEPALIVE within expected time frame")
		}

	case msg := <-client.msgsIn:
		switch msg.(type) {
		case *KeepaliveMessage:
			keepaliveMsg := *msg.(*KeepaliveMessage)
			client.keepaliveLast = time.Now()
			client.log().WithField("msg", keepaliveMsg).Debug("Received KEEPALIVE message")

		case *DataTransmissionMessage:
			dataTransMsg := *msg.(*DataTransmissionMessage)
			client.log().WithField("msg", dataTransMsg).Info("Received XFER_SEGMENT")

		case *SessionTerminationMessage:
			sesstermMsg := *msg.(*SessionTerminationMessage)
			client.log().WithField("msg", sesstermMsg).Info("Received SESS_TERM")
			return sessTermErr

		default:
			client.log().WithField("msg", msg).Warn("Received unexpected message")
		}

	case <-time.After(100 * time.Millisecond):
		// This case prevents blocking. Otherwise the select statement would wait
		// for the keepaliveTicker or an incoming message.
	}

	return nil
}

func (client *TCPCLClient) Send(bndl *bundle.Bundle) error {
	client.transferIdOut += 1
	var t = NewBundleTransfer(client.transferIdOut, *bndl)

	client.log().WithFields(log.Fields{
		"bundle":   bndl,
		"transfer": t,
	}).Info("Started Bundle Transfer")

	for {
		dtm, err := t.NextSegment(client.segmentMru)

		if err == io.EOF {
			client.log().Info("Finished Transfer")
			return nil
		} else if err != nil {
			client.log().WithError(err).Warn("Fetching Segment errored")
			return err
		}

		client.msgsOut <- &dtm
		client.log().WithField("msg", dtm).Debug("Sent XFER_SEGMENT")
	}
}
