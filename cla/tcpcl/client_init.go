package tcpcl

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// This file contains code for the Client's contact state.

// handleSessInit manges the initialization state.
func (client *TCPCLClient) handleSessInit() error {
	var logger = log.WithFields(log.Fields{
		"session": client,
		"state":   "initialization",
	})

	// XXX
	const (
		keepalive   = 10
		segmentMru  = 1500
		transferMru = 0xFFFFFFFF
	)

	switch {
	case client.active && !client.initSent, !client.active && !client.initSent && client.initRecv:
		client.sessInitSent = NewSessionInitMessage(keepalive, segmentMru, transferMru, client.endpointID.String())
		if err := client.sessInitSent.Marshal(client.rw); err != nil {
			return err
		} else if err := client.rw.Flush(); err != nil {
			return err
		} else {
			client.initSent = true
			logger.WithField("msg", client.sessInitSent).Debug("Sent SESS_INIT message")
		}

	case !client.active && !client.initRecv, client.active && client.initSent && !client.initRecv:
		if err := client.sessInitRecv.Unmarshal(client.rw); err != nil {
			return err
		} else {
			client.initRecv = true
			logger.WithField("msg", client.sessInitRecv).Debug("Received SESS_INIT message")
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

		logger.WithFields(log.Fields{
			"endpoint ID":  client.peerEndpointID,
			"keepalive":    client.keepalive,
			"segment MRU":  client.segmentMru,
			"transfer MRU": client.transferMru,
		}).Debug("Exchanged SESS_INIT messages")
		client.state.Next()
	}

	return nil
}
