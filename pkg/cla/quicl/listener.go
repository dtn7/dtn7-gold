// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package quicl

import (
	"context"
	"errors"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/cla/quicl/internal"
	"github.com/lucas-clemente/quic-go"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	listenAddress string
	endpointID    bpv7.EndpointID
	manager       *cla.Manager
	listener      quic.Listener
}

func NewQUICListener(listenAddress string, endpointID bpv7.EndpointID) *Listener {
	return &Listener{
		listenAddress: listenAddress,
		endpointID:    endpointID,
		manager:       nil,
		listener:      nil,
	}
}

/**
Methods for Convergable interface
*/

func (listener *Listener) Close() error {
	log.WithField("address", listener.listenAddress).Info("Shutting ourselves down")
	return listener.listener.Close()
}

/**
Methods for ConvergenceProvider interface
*/

func (listener *Listener) RegisterManager(manager *cla.Manager) {
	listener.manager = manager
}

func (listener *Listener) Start() error {
	log.WithField("address", listener.listenAddress).Info("Starting QUICL-listener")
	lst, err := quic.ListenAddr(listener.listenAddress, internal.GenerateSimpleListenerTLSConfig(), internal.GenerateQUICConfig())
	if err != nil {
		log.WithError(err).Error("Error creating QUICL listener")
		return err
	}

	listener.listener = lst
	go listener.handle()

	return nil
}

/*
Non-interface methods
*/

func (listener *Listener) handle() {
	log.WithField("address", listener.listenAddress).Info("Listening for QUICL connections")

	for {
		session, err := listener.listener.Accept(context.Background())
		if err != nil {
			if !(errors.Is(err, context.DeadlineExceeded)) {
				if err.Error() == "quic: Server closed" {
					log.WithField("address", listener.listenAddress).Info("Shutting this place down")
					return
				}

				log.WithFields(log.Fields{
					"address": listener.listenAddress,
					"error":   err,
				}).Error("Unknown error accepting QUIC connection")
			}
		} else {
			log.WithFields(log.Fields{
				"address": listener.listenAddress,
				"peer":    session.RemoteAddr(),
			}).Info("QUICL listener accepted new connection")
			endpoint := NewListenerEndpoint(listener.endpointID, session)
			go listener.manager.Register(endpoint)
		}
	}
}
