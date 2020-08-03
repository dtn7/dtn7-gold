// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// Connector implements both a cla.ConvergenceReceiver and cla.ConvergenceSender and supplies the possibility
// to receive and send Bundles over a Modem. However, based on the broadcasting nature of this CLA, addressing
// specific recipients is not possible. Furthermore, attributing senders is also not possible.
type Connector struct {
	modem            Modem
	permanent        bool
	tid              byte
	transmissions    map[byte]*IncomingTransmission
	fragmentOut      chan Fragment
	failTransmission chan byte
	reportChan       chan cla.ConvergenceStatus

	closedRSyn chan struct{}
	closedRAck chan struct{}
	closedWSyn chan struct{}
	closedWAck chan struct{}
}

// NewConnector creates a new Connector, wrapping around the given Modem.
func NewConnector(modem Modem, permanent bool) *Connector {
	return &Connector{
		modem:            modem,
		permanent:        permanent,
		tid:              randomTransmissionId(),
		transmissions:    make(map[byte]*IncomingTransmission),
		fragmentOut:      make(chan Fragment, 64),
		failTransmission: make(chan byte, 64),
		reportChan:       make(chan cla.ConvergenceStatus, 64),
	}
}

func (c *Connector) Start() (error, bool) {
	c.closedRSyn = make(chan struct{})
	c.closedRAck = make(chan struct{})
	c.closedWSyn = make(chan struct{})
	c.closedWAck = make(chan struct{})

	go c.handlerRead()
	go c.handlerWrite()

	return nil, false
}

// handlerRead acts on incoming Fragments.
func (c *Connector) handlerRead() {
	defer close(c.closedRAck)

	var logger = log.WithField("bbc", c.Address())

	for {
		select {
		case <-c.closedRSyn:
			logger.Info("Received close signal, stopping handlerRead")
			return

		default:
			if frag, err := c.modem.Receive(); err == io.EOF {
				logger.Info("Read EOF, stopping handlerRead")
				return
			} else if err != nil {
				logger.WithError(err).Warn("Receiving Fragments from Modem errored")
			} else if err = c.handleIncomingFragment(frag); err != nil {
				logger.WithError(err).Warn("Handling incoming Fragment errored")
			}
		}
	}
}

// handleIncomingFragment inspects a new Fragment and tries to add it to a new or a known Transmission.
func (c *Connector) handleIncomingFragment(frag Fragment) (err error) {
	var (
		logger = log.WithField("bbc", c.Address())

		transmission *IncomingTransmission
		known        bool
	)

	defer func() {
		// Report a failed Transmission in case of an error.
		if err == nil {
			return
		}

		c.fragmentOut <- frag.ReportFailure()
		logger.WithField("fragment", frag).Info("Broadcasting failure Fragment")
	}()

	if frag.FailBit() {
		c.failTransmission <- frag.TransmissionID()
		return
	}

	if transmission, known = c.transmissions[frag.TransmissionID()]; !known {
		transmission, err = c.handleIncomingNewTransmission(frag)
	} else {
		err = c.handleIncomingKnownTransmission(frag, transmission)
	}
	if err != nil {
		logger.WithError(err).WithField("transmission", transmission).Warn(
			"Fetching or creating Transmission errored")

		return
	}

	if transmission.IsFinished() {
		var bndl bundle.Bundle

		if bndl, err = transmission.Bundle(); err == nil {
			logger.WithFields(log.Fields{
				"transaction": transmission,
				"bundle":      bndl.ID(),
			}).Info("Bundle Broadcasting Connector received Bundle")

			c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
		} else {
			// Returning error variable err keeps its value and cleanup code follows. That's why we don't return here.
			logger.WithError(err).WithField("transmission", transmission).Warn(
				"Extracting Bundle from Transmission errored")
		}

		delete(c.transmissions, transmission.TransmissionID)
	}
	return
}

// handleIncomingNewTransmission creates a new Transmission for a Fragment with an unknown Transmission ID.
func (c *Connector) handleIncomingNewTransmission(frag Fragment) (trans *IncomingTransmission, err error) {
	if trans, err = NewIncomingTransmission(frag); err == nil {
		c.transmissions[trans.TransmissionID] = trans
	}
	return
}

// handleIncomingKnownTransmission updates a known Transmission with the next Fragment.
func (c *Connector) handleIncomingKnownTransmission(frag Fragment, trans *IncomingTransmission) (err error) {
	if _, err = trans.ReadFragment(frag); err != nil {
		delete(c.transmissions, trans.TransmissionID)
	}
	return
}

// handlerWrite acts on outgoing Fragments.
func (c *Connector) handlerWrite() {
	defer close(c.closedWAck)

	var logger = log.WithField("bbc", c.Address())

	for {
		select {
		case <-c.closedWSyn:
			logger.Info("Received close signal, stopping handlerWrite")
			return

		case f := <-c.fragmentOut:
			if err := c.modem.Send(f); err != nil {
				logger.WithField("fragment", f).WithError(err).Warn("Transmitting Fragment errored")
			}
		}
	}
}

func (c *Connector) Close() {
	close(c.closedRSyn)
	close(c.closedWSyn)

	if err := c.modem.Close(); err != nil {
		log.WithField("bbc", c.Address()).WithError(err).Warn("Closing Modem errored")
	}

	<-c.closedRAck
	<-c.closedWAck
}

func (c *Connector) Channel() chan cla.ConvergenceStatus {
	return c.reportChan
}

func (c *Connector) Address() string {
	return fmt.Sprintf("bbc://%v", c.modem)
}

func (c *Connector) IsPermanent() bool {
	return c.permanent
}

func (c *Connector) Send(bndl *bundle.Bundle) error {
	var t, tErr = NewOutgoingTransmission(c.tid, *bndl, c.modem.Mtu())
	if tErr != nil {
		return tErr
	}

	c.tid = nextTransmissionId(c.tid)

	for {
		select {
		case failedTransmissionId := <-c.failTransmission:
			if failedTransmissionId == t.TransmissionID {
				log.WithField("bbc", c.Address()).Warn("Received failure Fragment")
				return fmt.Errorf("peer send failure Fragment")
			}

		default:
			f, fin, err := t.WriteFragment()
			logger := log.WithFields(log.Fields{
				"bbc":      c.Address(),
				"fragment": f,
			})

			if err != nil {
				logger.WithError(err).Warn("Creating Fragment errored")
				return err
			}

			c.fragmentOut <- f
			if fin {
				logger.Debug("Transmitted last Fragment")
				return nil
			}
		}
	}
}

func (c *Connector) GetPeerEndpointID() bundle.EndpointID {
	return bundle.DtnNone()
}

func (c *Connector) GetEndpointID() bundle.EndpointID {
	return bundle.DtnNone()
}

func (c *Connector) String() string {
	return c.Address()
}
