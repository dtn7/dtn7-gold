// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package quicl

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/cla/quicl/internal"
	"github.com/lucas-clemente/quic-go"
	log "github.com/sirupsen/logrus"
)

const handshakeTimeout = 500 * time.Millisecond

type Endpoint struct {
	id          bpv7.EndpointID
	peerId      bpv7.EndpointID
	peerAddress string
	peerHost    string
	connection  quic.Connection

	reportingChannel chan cla.ConvergenceStatus

	permanent bool
	dialer    bool
}

func NewListenerEndpoint(id bpv7.EndpointID, session quic.Connection) *Endpoint {
	peerHost, _, _ := net.SplitHostPort(session.RemoteAddr().String())

	return &Endpoint{
		id:               id,
		peerAddress:      session.RemoteAddr().String(),
		peerHost:         peerHost,
		connection:       session,
		reportingChannel: make(chan cla.ConvergenceStatus),
		permanent:        false,
		dialer:           false,
	}
}

func NewDialerEndpoint(peerAddress string, id bpv7.EndpointID, permanent bool) *Endpoint {
	peerHost, _, err := net.SplitHostPort(peerAddress)
	if err != nil {
		log.WithFields(log.Fields{
			"address": peerAddress,
			"error":   err,
		}).Fatal("Invalid peer address")
	}

	return &Endpoint{
		id:               id,
		peerAddress:      peerAddress,
		peerHost:         peerHost,
		reportingChannel: make(chan cla.ConvergenceStatus),
		permanent:        permanent,
		dialer:           true,
	}
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("QUICLEndpoint{Peer ID: %v, Peer Address: %v, Dialer: %v, Permanent: %v}", endpoint.peerId, endpoint.peerAddress, endpoint.dialer, endpoint.permanent)
}

/**
Methods for Convergable interface
*/

func (endpoint *Endpoint) Close() error {
	log.WithField("peer", endpoint.peerAddress).Debug("Someone called Close()")
	err := endpoint.connection.CloseWithError(internal.ApplicationShutdown, "Daemon shutting down")
	return err
}

/**
Methods for Convergence interface
*/

func (endpoint *Endpoint) Start() (error, bool) {
	// if we are on the dialer-side we need to first initiate the quic-connection
	if endpoint.dialer {
		session, err := quic.DialAddr(endpoint.peerAddress, internal.GenerateDialerTLSConfig(), internal.GenerateQUICConfig())
		endpoint.connection = session
		if err != nil {
			return err, endpoint.permanent
		}
	}

	log.WithFields(log.Fields{
		"endpoint": endpoint.id,
		"peer":     endpoint.peerAddress,
	}).Debug("Starting CLA")

	var err error
	if endpoint.dialer {
		err = endpoint.handshakeDialer()
	} else {
		err = endpoint.handshakeListener()
	}

	if err != nil {
		var herr *internal.HandshakeError
		if errors.As(err, &herr) {
			log.WithFields(log.Fields{
				"cla":      endpoint,
				"error":    herr,
				"internal": herr.Unwrap(),
			}).Warn("Handshake failure")
			_ = endpoint.connection.CloseWithError(herr.Code, herr.Msg)
		} else {
			log.WithFields(log.Fields{
				"cla":   endpoint,
				"error": err,
			}).Error("Non handshake-related error during handshake")
			_ = endpoint.connection.CloseWithError(internal.LocalError, "Local error")
		}
		return err, endpoint.permanent
	} else {
		go endpoint.handleConnection()
	}

	return err, endpoint.permanent
}

func (endpoint *Endpoint) Channel() chan cla.ConvergenceStatus {
	return endpoint.reportingChannel
}

func (endpoint *Endpoint) Address() string {
	// we return only the host, since connections are bidirectional
	// ,and we don't want to open a new connection on a different port to the same peer
	return endpoint.peerHost
}

func (endpoint *Endpoint) IsPermanent() bool {
	return endpoint.permanent
}

/**
Methods for ConvergenceReceiver interface
*/

func (endpoint *Endpoint) GetEndpointID() bpv7.EndpointID {
	return endpoint.id
}

/**
Methods for ConvergenceSender interface
*/

func (endpoint *Endpoint) GetPeerEndpointID() bpv7.EndpointID {
	return endpoint.peerId
}

func (endpoint *Endpoint) Send(bndl bpv7.Bundle) error {
	stream, err := endpoint.connection.OpenStream()
	if err != nil {
		// TODO: understand possible error cases
		return err
	}

	buff := new(bytes.Buffer)
	if err = cboring.Marshal(&bndl, buff); err != nil {
		stream.CancelWrite(internal.DataMarshalError)
		_ = stream.Close()
		return err
	}

	// TODO: Do we actually need the bufio-wrapper?
	writer := bufio.NewWriter(stream)
	if _, err = buff.WriteTo(writer); err != nil {
		stream.CancelWrite(internal.StreamTransmissionError)
		_ = stream.Close()
		return err
	}

	if err = writer.Flush(); err != nil {
		stream.CancelWrite(internal.StreamTransmissionError)
		_ = stream.Close()
		return err
	}

	_ = stream.Close()
	return nil
}

/*
Non-interface methods
*/

func (endpoint *Endpoint) handleConnection() {
	for {
		stream, err := endpoint.connection.AcceptStream(context.Background())
		if err != nil {
			var netErr net.Error
			var appErr *quic.ApplicationError

			switch {
			case errors.As(err, &netErr):
				if netErr.Timeout() {
					log.WithFields(log.Fields{
						"CLA":   endpoint,
						"error": netErr,
					}).Debug("Peer timed out.")

					endpoint.reportPeerDisappeared()

					return
				}

			case errors.As(err, &appErr):
				log.WithFields(log.Fields{
					"peer":       endpoint.peerId,
					"remote":     appErr.Remote,
					"error code": appErr.ErrorCode,
					"error msg":  appErr.ErrorMessage,
				}).Debug("Connection to peer closed")
				if appErr.Remote {
					endpoint.reportPeerDisappeared()
				}
				return

			default:
				log.WithFields(log.Fields{
					"CLA":   endpoint,
					"error": err,
				}).Error("Unexpected error while waiting for stream")
			}
		} else {
			go endpoint.handleStream(stream)
		}
	}
}

func (endpoint *Endpoint) handleStream(stream quic.Stream) {
	log.WithField("cla", endpoint).Debug("Receiving bundle via quicl")

	// TODO: Do we actually need the bufio-wrapper?
	reader := bufio.NewReader(stream)

	bundle := new(bpv7.Bundle)
	if err := cboring.Unmarshal(bundle, reader); err != nil {
		log.WithFields(log.Fields{
			"cla":   endpoint,
			"error": err,
		}).Error("quicl failed to read bundle")

		stream.CancelRead(internal.StreamTransmissionError)
	} else {
		log.WithFields(log.Fields{
			"cla": endpoint,
		}).Debug("quicl received a bundle")

		endpoint.reportingChannel <- cla.NewConvergenceReceivedBundle(endpoint, endpoint.id, bundle)
	}
	log.WithField("cla", endpoint).Debug("Finished handling stream")
}

func (endpoint *Endpoint) handshakeListener() error {
	log.WithField("cla", endpoint.peerAddress).Debug("Performing handshake")

	// the dialer has half a second to initiate the handshake
	ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
	defer cancel()

	// wait for the dialer to open a stream
	stream, err := endpoint.connection.AcceptStream(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return internal.NewHandshakeError("dialer took too long to initiate handshake", internal.PeerError, err)
		} else {
			return internal.NewHandshakeError("unanticipated error happened", internal.UnknownError, err)
		}
	}

	// the listener first receives the dialer's id
	if err = endpoint.receiveEndpointID(stream); err != nil {
		return err
	}

	// then send our own id
	if err = endpoint.sendEndpointID(stream); err != nil {
		return err
	}

	// lastly, close the stream
	if err = stream.Close(); err != nil {
		return internal.NewHandshakeError("error closing handshake stream", internal.ConnectionError, err)
	}

	return nil
}

func (endpoint *Endpoint) handshakeDialer() error {
	log.WithField("cla", endpoint.peerAddress).Debug("Performing handshake")

	stream, err := endpoint.connection.OpenStream()
	if err != nil {
		return internal.NewHandshakeError("Error during stream initiation", internal.ConnectionError, err)
	}

	// start by sending own ID
	err = endpoint.sendEndpointID(stream)
	if err != nil {
		return err
	}

	// wait for our peer's ID
	err = endpoint.receiveEndpointID(stream)

	return err
}

func (endpoint *Endpoint) sendEndpointID(stream quic.Stream) error {
	log.WithField("cla", endpoint).Debug("Sending own endpoint id")

	buff := new(bytes.Buffer)
	if err := cboring.Marshal(&endpoint.id, buff); err != nil {
		return internal.NewHandshakeError("error marshaling endpoint-id", internal.LocalError, err)
	}

	// TODO: Do we actually need the bufio-wrapper?
	writer := bufio.NewWriter(stream)
	if err := cboring.WriteByteStringLen(uint64(buff.Len()), writer); err != nil {
		return internal.NewHandshakeError("error sending id length", internal.ConnectionError, err)
	}

	if _, err := buff.WriteTo(writer); err != nil {
		return internal.NewHandshakeError("error sending id", internal.ConnectionError, err)
	}

	if err := writer.Flush(); err != nil {
		return internal.NewHandshakeError("error flushing write-buffer", internal.ConnectionError, err)
	}

	return nil
}

func (endpoint *Endpoint) receiveEndpointID(stream quic.Stream) error {
	log.WithField("cla", endpoint).Debug("Receiving peer's endpoint id")
	reader := bufio.NewReader(stream)

	length, err := cboring.ReadByteStringLen(reader)
	if err != nil && !errors.Is(err, io.EOF) {
		return internal.NewHandshakeError("error reading id length", internal.ConnectionError, err)
	} else if length == 0 {
		return internal.NewHandshakeError("error reading id length", internal.ConnectionError, fmt.Errorf("length is 0"))
	}

	id := new(bpv7.EndpointID)
	if err = cboring.Unmarshal(id, reader); err != nil {
		// TODO: distinguish cbor and transmission errors
		return internal.NewHandshakeError("error reading id", internal.ConnectionError, err)
	}

	log.WithFields(log.Fields{
		"cla":     endpoint,
		"peer id": id,
	}).Debug("Received peer's endpoint id")

	endpoint.peerId = *id

	return nil
}

func (endpoint *Endpoint) reportPeerDisappeared() {
	endpoint.reportingChannel <- cla.NewConvergencePeerDisappeared(endpoint, endpoint.peerId)
}
