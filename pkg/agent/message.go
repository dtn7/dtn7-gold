// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// Message is a generic interface to specify an information exchange between an ApplicationAgent and some Manager.
// The following types named *Message are implementations of this interface.
type Message interface {
	// Recipients returns a list of endpoints to which this message is addressed.
	// However, if this message is not addressed to some specific endpoint, nil must be returned.
	Recipients() []bpv7.EndpointID
}

// BundleMessage indicates a transmitted Bundle.
// If the Message is received from an ApplicationAgent, it is an incoming Bundle.
// If the Message is sent from an ApplicationAgent, it is an outgoing Bundle.
type BundleMessage struct {
	Bundle bpv7.Bundle
}

// Recipients are the Bundle destination for a BundleMessage.
func (bm BundleMessage) Recipients() []bpv7.EndpointID {
	return []bpv7.EndpointID{bm.Bundle.PrimaryBlock.Destination}
}

// SyscallRequestMessage is sent from an ApplicationAgent to request some "syscall" specific information.
type SyscallRequestMessage struct {
	Sender  bpv7.EndpointID
	Request string
}

// Recipients are not available for a SyscallRequestMessage.
func (srm SyscallRequestMessage) Recipients() []bpv7.EndpointID {
	return []bpv7.EndpointID{srm.Sender}
}

// SyscallResponseMessage is the answer to a SyscallRequestMessage, sent to an ApplicationAgent.
// The Response is stored as a generic byte array. However, its content is defined for each syscall.
type SyscallResponseMessage struct {
	Request   string
	Response  []byte
	Recipient bpv7.EndpointID
}

// Recipients are the sender of the SyscallRequestMessage.
func (srm SyscallResponseMessage) Recipients() []bpv7.EndpointID {
	return []bpv7.EndpointID{srm.Recipient}
}

// ShutdownMessage indicates the closing down of an ApplicationAgent.
// If the Message is received from an ApplicationAgent, it must close itself down.
// If the Message is sent from an ApplicationAgent, it is closing down itself.
type ShutdownMessage struct{}

// Recipients are not available for a ShutdownMessage.
func (sm ShutdownMessage) Recipients() []bpv7.EndpointID {
	return nil
}
