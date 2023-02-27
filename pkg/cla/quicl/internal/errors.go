// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import "github.com/quic-go/quic-go"

const (
	// UnknownError is the catchall error code for things I didn't foresee
	// Why is this necessary? Because the quic package isn't terribly concerned with documenting its error states...
	UnknownError quic.ApplicationErrorCode = 1
	// ApplicationShutdown is sent when the daemon is shut down and terminates its connections
	// Why is this necessary? Because the quic package wants there to be an error code when a connection is closed
	// There does not appear to by any notion of "normal" termination
	ApplicationShutdown quic.ApplicationErrorCode = 2
	// LocalError designates errors that happen on this machine (like failing to marshall a data structure)
	LocalError quic.ApplicationErrorCode = 3
	// PeerError designates a failure-state where the remote peer has violated the protocol
	PeerError quic.ApplicationErrorCode = 4
	// ConnectionError designates errors in underlying data transmission
	ConnectionError quic.ApplicationErrorCode = 5

	DataMarshalError        quic.StreamErrorCode = 1
	StreamTransmissionError quic.StreamErrorCode = 2
)

// HandshakeError is thrown by either the listener or dialer if there is any problem during the protocol handshake
// If the issue was an error thrown by a library function, the Cause-field will wrap the instigating error
type HandshakeError struct {
	Msg   string
	Code  quic.ApplicationErrorCode
	Cause error
}

func NewHandshakeError(message string, code quic.ApplicationErrorCode, cause error) *HandshakeError {
	return &HandshakeError{
		Msg:   message,
		Code:  code,
		Cause: cause,
	}
}

func (err *HandshakeError) Error() string {
	return err.Msg
}

func (err *HandshakeError) Unwrap() error {
	return err.Cause
}

type InitialisationError struct {
	Msg string
}

func NewInitialisationError(message string) *InitialisationError {
	return &InitialisationError{
		Msg: message,
	}
}

func (err *InitialisationError) Error() string {
	return err.Msg
}
