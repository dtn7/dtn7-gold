// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/agent"
	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// pinger manages to send ping bundles and show their acknowledgment.
type pinger struct {
	sender   string
	receiver string

	websocketConn *agent.WebSocketAgentConnector

	closeChan      chan os.Signal
	bundleReadChan chan bpv7.Bundle
}

// pingBundle creates a ping bundle.
func (p *pinger) pingBundle() (bpv7.Bundle, error) {
	return bpv7.Builder().
		CRC(bpv7.CRC32).
		Source(p.sender).
		Destination(p.receiver).
		BundleCtrlFlags(bpv7.MustNotFragmented).
		CreationTimestampNow().
		Lifetime("1m").
		HopCountBlock(64).
		PayloadBlock([]byte("ping")).
		Build()
}

// handleBundleRead forwards received bundles to the main handle function.
func (p *pinger) handleBundleRead() {
	for {
		if b, err := p.websocketConn.ReadBundle(); err != nil {
			log.WithError(err).Error("Reading Bundle errored")

			close(p.bundleReadChan)
			return
		} else {
			p.bundleReadChan <- b
		}
	}
}

// handle a pinger's task.
func (p *pinger) handle() {
	ticker := time.NewTicker(time.Second)

	defer p.websocketConn.Close()
	defer ticker.Stop()

	for {
		select {
		case <-p.closeChan:
			return

		case <-ticker.C:
			if b, err := p.pingBundle(); err != nil {
				log.WithError(err).Error("Cannot create ping bundle")
			} else if err := p.websocketConn.WriteBundle(b); err != nil {
				log.WithError(err).Error("Cannot send ping bundle")
			} else {
				log.Info("Sent ping bundle")
			}

		case b, ok := <-p.bundleReadChan:
			if !ok {
				log.Error("Bundle reader channel was closed")
				return
			}

			log.WithField("bundle", b).Info("Received bundle")
		}
	}
}

// ping another dtn host
func ping(args []string) {
	if len(args) != 3 {
		printUsage()
	}

	p := pinger{
		sender:         args[1],
		receiver:       args[2],
		closeChan:      make(chan os.Signal),
		bundleReadChan: make(chan bpv7.Bundle),
	}

	var err error
	if p.websocketConn, err = agent.NewWebSocketAgentConnector(args[0], p.sender); err != nil {
		printFatal(err, "Starting WebSocketAgentConnector errored")
	}

	signal.Notify(p.closeChan, os.Interrupt)

	go p.handleBundleRead()
	p.handle()
}
