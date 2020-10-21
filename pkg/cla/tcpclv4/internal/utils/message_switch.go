// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"io"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// MessageSwitch is the interface for a exchange between msgs.Message from channels and an underlying layer.
type MessageSwitch interface {
	io.Closer

	// Exchange channels to be serialized.
	//
	// 	* incoming is a "receive only" channel for incoming Messages.
	//	* outgoing is a "send only" channel for outgoing Messages.
	//	* errChan is another "receive only" channel to propagate errors. Only one error should be sent.
	Exchange() (incoming <-chan msgs.Message, outgoing chan<- msgs.Message, errChan <-chan error)
}
