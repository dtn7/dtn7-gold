// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// PingAgent is a simple ApplicationAgent to "pong" / acknowledge incoming Bundles.
type PingAgent struct {
	endpoint bundle.EndpointID
	receiver chan Message
	sender   chan Message
}

// NewPing creates a new PingAgent ApplicationAgent.
func NewPing(endpoint bundle.EndpointID) *PingAgent {
	p := &PingAgent{
		endpoint: endpoint,
		receiver: make(chan Message),
		sender:   make(chan Message),
	}

	go p.handler()

	return p
}

func (p *PingAgent) log() *log.Entry {
	return log.WithField("PingAgent", p.endpoint)
}

func (p *PingAgent) handler() {
	defer close(p.sender)

	for m := range p.receiver {
		switch m := m.(type) {
		case BundleMessage:
			p.ackBundle(m.Bundle)

		case ShutdownMessage:
			return

		default:
			p.log().WithField("message", m).Info("Received unsupported Message")
		}
	}
}

func (p *PingAgent) ackBundle(b bundle.Bundle) {
	bndl, err := bundle.Builder().
		Source(p.endpoint).
		Destination(b.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime("24h").
		HopCountBlock(64).
		PayloadBlock([]byte("pong")).
		Build()

	if err != nil {
		p.log().WithError(err).Warn("Building ACK Bundle errored")
	} else {
		p.log().WithField("bundle", bndl).Info("Sending ACK Bundle")
		p.sender <- BundleMessage{bndl}
	}
}

func (p *PingAgent) Endpoints() []bundle.EndpointID {
	return []bundle.EndpointID{p.endpoint}
}

func (p *PingAgent) MessageReceiver() chan Message {
	return p.receiver
}

func (p *PingAgent) MessageSender() chan Message {
	return p.sender
}
