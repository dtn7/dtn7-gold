// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import "github.com/lucas-clemente/quic-go"

const (
	// UnknownError is the catchall error code for things I didn't foresee
	// Why is this necessary? Because the quic package isn't terribly concerned with documenting its error states...
	UnknownError quic.ApplicationErrorCode = 1
	// LocalError designates errors that happen on this machine (like failing to marshall a data structure)
	LocalError quic.ApplicationErrorCode = 2
	// ConnectionError designates errors in data transmission
	ConnectionError quic.ApplicationErrorCode = 3
	PeerError       quic.ApplicationErrorCode = 4
	// ApplicationShutdown is sent when the deamon is shut down and terminates its connections
	ApplicationShutdown quic.ApplicationErrorCode = 5

	DataMarshalError        quic.StreamErrorCode = 1
	StreamTransmissionError quic.StreamErrorCode = 2
)

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
