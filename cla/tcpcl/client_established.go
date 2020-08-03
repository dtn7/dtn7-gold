// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// This file contains code for the Client's established state.

// handleEstablished manges the established state.
func (client *Client) handleEstablished() (err error) {
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
		switch msg := msg.(type) {
		case *KeepaliveMessage:
			keepaliveMsg := *msg
			client.keepaliveLast = time.Now()
			client.log().WithField("msg", keepaliveMsg).Debug("Received KEEPALIVE message")

		case *DataTransmissionMessage:
			dtm := *msg
			client.log().WithField("msg", dtm).Debug("Received XFER_SEGMENT")

			if client.transferIn != nil && dtm.Flags&SegmentStart != 0 {
				client.log().WithField("msg", dtm).Warn(
					"Received XFER_SEGMENT with START flag, but has old transfer; resetting")

				client.transferIn = NewIncomingTransfer(dtm.TransferId)
			} else if client.transferIn == nil {
				if dtm.Flags&SegmentStart == 0 {
					client.log().WithField("msg", dtm).Warn(
						"Received XFER_SEGMENT without a START flag, but no transfer state")

					ackMsg := NewTransferRefusalMessage(RefusalUnknown, dtm.TransferId)
					client.msgsOut <- &ackMsg
				} else {
					client.log().WithField("msg", dtm).Debug("Create new incoming transfer")

					client.transferIn = NewIncomingTransfer(dtm.TransferId)
				}
			}

			if client.transferIn != nil {
				if dam, err := client.transferIn.NextSegment(dtm); err != nil {
					client.log().WithError(err).WithField("msg", dtm).Warn(
						"Parsing next incoming segment errored")

					ackMsg := NewTransferRefusalMessage(RefusalUnknown, dtm.TransferId)
					client.msgsOut <- &ackMsg
				} else {
					client.msgsOut <- &dam
					client.log().WithField("msg", dam).Debug("Sent XFER_ACK")
				}

				if client.transferIn.IsFinished() {
					client.log().WithField("transfer", client.transferIn).Info(
						"Finished incoming transfer")

					if bndl, err := client.transferIn.ToBundle(); err != nil {
						client.log().WithError(err).WithField("transfer", client.transferIn).Warn(
							"Unmarshalling Bundle errored")
					} else {
						client.log().WithFields(log.Fields{
							"transfer": client.transferIn,
							"bundle":   bndl,
						}).Info("Received Bundle")

						client.reportChan <- cla.NewConvergenceReceivedBundle(
							client, client.endpointID, &bndl)
					}

					client.transferIn = nil
				}
			}

		case *DataAcknowledgementMessage, *TransferRefusalMessage:
			client.transferOutAck <- msg

		case *SessionTerminationMessage:
			sesstermMsg := *msg
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

func (client *Client) Send(bndl *bundle.Bundle) error {
	client.transferOutMutex.Lock()
	defer client.transferOutMutex.Unlock()

	if !client.state.IsEstablished() {
		return fmt.Errorf("Client is not in an established state")
	}

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
		switch ackMsg := ackMsg.(type) {
		case *DataAcknowledgementMessage:
			tlog.WithField("msg", ackMsg).Debug("Received XFER_ACK")

			if ackMsg.TransferId != dtm.TransferId || ackMsg.Flags != dtm.Flags {
				tlog.WithField("msg", ackMsg).Warn("XFER_ACK does not match XFER_SEGMENT")
				return fmt.Errorf("XFER_ACK does not match XFER_SEGMENT")
			}

		case *TransferRefusalMessage:
			tlog.WithField("msg", ackMsg).Warn("Received XFER_REFUSE, aborting transfer")
			return fmt.Errorf("Received XFER_REFUSE, aborting transfer")

		default:
			tlog.WithField("msg", ackMsg).Warn("Received wrong message type")
			return fmt.Errorf("Received wrong message type")
		}
	}
}
