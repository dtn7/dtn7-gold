// SPDX-FileCopyrightText: 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// PingAgent is a simple ApplicationAgent to "pong" / acknowledge incoming Bundles.
type PingAgent struct {
	endpoint bpv7.EndpointID
	receiver chan Message
	sender   chan Message
}

// NewPing creates a new PingAgent ApplicationAgent.
func NewPing(endpoint bpv7.EndpointID) *PingAgent {
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

func (p *PingAgent) ackBundle(b bpv7.Bundle) {
	hopCount := 64
	if hc, err := b.ExtensionBlock(bpv7.ExtBlockTypeHopCountBlock); err == nil {
		hopCount = int(hc.Value.(*bpv7.HopCountBlock).Limit)
	}

	bndl, err := bpv7.Builder().
		CRC(bpv7.CRC32).
		Source(p.endpoint).
		Destination(b.PrimaryBlock.ReportTo).
		BundleCtrlFlags(bpv7.MustNotFragmented).
		CreationTimestampNow().
		Lifetime(b.PrimaryBlock.Lifetime).
		HopCountBlock(hopCount).
		PayloadBlock([]byte("pong")).
		Build()

	if err != nil {
		p.log().WithError(err).Warn("Building ACK Bundle errored")
	} else {
		p.log().WithField("bundle", bndl).Info("Sending ACK Bundle")
		p.sender <- BundleMessage{bndl}
	}
}

func (p *PingAgent) Endpoints() []bpv7.EndpointID {
	return []bpv7.EndpointID{p.endpoint}
}

func (p *PingAgent) MessageReceiver() chan Message {
	return p.receiver
}

func (p *PingAgent) MessageSender() chan Message {
	return p.sender
}
