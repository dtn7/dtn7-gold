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
			client.log().WithField("msg", dataTransMsg).Debug("Received XFER_SEGMENT")

			// TODO: create correct ACK
			ackMsg := NewDataAcknowledgementMessage(dataTransMsg.Flags, dataTransMsg.TransferId, 0)
			client.msgsOut <- &ackMsg
			client.log().WithField("msg", ackMsg).Debug("Sent XFER_ACK")

		case *DataAcknowledgementMessage, *TransferRefusalMessage:
			client.transferOutAck <- msg

		case *SessionTerminationMessage:
			sesstermMsg := *msg.(*SessionTerminationMessage)
			client.log().WithField("msg", sesstermMsg).Info("Received SESS_TERM")
			return sessTermErr

		default:
			client.log().WithField("msg", msg).Warn("Received unexpected message")
		}

	case msg := <-client.transferOutSend:
		if _, ok := msg.(*DataTransmissionMessage); !ok {
			client.log().WithField("msg", msg).Warn(
				"Transfer Out received a non XFER_SEGMENT message")
		} else {
			client.msgsOut <- msg
			client.log().WithField("msg", msg).Debug("Forwarded XFER_SEGMENT")
		}

	case <-time.After(time.Millisecond):
		// This case prevents blocking. Otherwise the select statement would wait
		// for the keepaliveTicker or an incoming message.
	}

	return nil
}

func (client *TCPCLClient) Send(bndl *bundle.Bundle) error {
	client.transferOutMutex.Lock()
	defer client.transferOutMutex.Unlock()

	client.transferOutId += 1
	var t = NewBundleOutgoingTransfer(client.transferOutId, *bndl)

	var tlog = client.log().WithFields(log.Fields{
		"bundle":   bndl,
		"transfer": t,
	})
	tlog.Info("Started Bundle Transfer")

	for {
		dtm, err := t.NextSegment(client.segmentMru)

		if err == io.EOF {
			tlog.Info("Finished Transfer")
			return nil
		} else if err != nil {
			tlog.WithError(err).Warn("Fetching Segment errored")
			return err
		}

		client.transferOutSend <- &dtm
		tlog.WithField("msg", dtm).Debug("Send disposed XFER_SEGMENT")

		ackMsg := <-client.transferOutAck
		switch ackMsg.(type) {
		case *DataAcknowledgementMessage:
			tlog.WithField("msg", ackMsg).Debug("Received XFER_ACK")
			// TODO: insepct ack

		case *TransferRefusalMessage:
			tlog.WithField("msg", ackMsg).Warn("Received XFER_REFUSE, aborting transfer")
			return fmt.Errorf("Received XFER_REFUSE, aborting transfer")

		default:
			tlog.WithField("msg", ackMsg).Warn("Received wrong message type")
			return fmt.Errorf("Received wrong message type")
		}
	}
}
