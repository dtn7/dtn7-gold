package tcpcl

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// This file contains code for the Client's contact state.

// handleSessInit manges the initialization state.
func (client *TCPCLClient) handleSessInit() error {
	// XXX
	const (
		keepalive   = 10
		segmentMru  = 1500
		transferMru = 0xFFFFFFFF
	)

	switch {
	case client.active && !client.initSent, !client.active && !client.initSent && client.initRecv:
		client.sessInitSent = NewSessionInitMessage(keepalive, segmentMru, transferMru, client.endpointID.String())
		client.initSent = true

		client.msgsOut <- &client.sessInitSent
		client.log().WithField("msg", client.sessInitSent).Debug("Sent SESS_INIT message")

	case !client.active && !client.initRecv, client.active && client.initSent && !client.initRecv:
		msg := <-client.msgsIn
		switch msg.(type) {
		case *SessionInitMessage:
			client.sessInitRecv = *msg.(*SessionInitMessage)
			client.initRecv = true
			client.log().WithField("msg", client.sessInitRecv).Debug("Received SESS_INIT message")

		default:
			client.log().WithField("msg", msg).Warn("Received wrong message")
			return fmt.Errorf("Wrong message type")
		}

	case client.initSent && client.initRecv:
		if eid, err := bundle.NewEndpointID(client.sessInitRecv.Eid); err != nil {
			return err
		} else {
			client.peerEndpointID = eid
		}

		client.keepalive = client.sessInitSent.KeepaliveInterval
		if client.sessInitRecv.KeepaliveInterval < client.keepalive {
			client.keepalive = client.sessInitRecv.KeepaliveInterval
		}
		client.segmentMru = client.sessInitSent.SegmentMru
		if client.sessInitRecv.SegmentMru < client.segmentMru {
			client.segmentMru = client.sessInitRecv.SegmentMru
		}
		client.transferMru = client.sessInitSent.TransferMru
		if client.sessInitRecv.TransferMru < client.transferMru {
			client.transferMru = client.sessInitRecv.TransferMru
		}

		client.log().WithFields(log.Fields{
			"endpoint ID":  client.peerEndpointID,
			"keepalive":    client.keepalive,
			"segment MRU":  client.segmentMru,
			"transfer MRU": client.transferMru,
		}).Debug("Exchanged SESS_INIT messages")
		client.state.Next()
	}

	return nil
}
