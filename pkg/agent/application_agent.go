// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import "github.com/dtn7/dtn7-go/pkg/bpv7"

// ApplicationAgent is an interface to describe application agents, which can both receive and transmit Bundles.
// Each implementation must provide the following methods to communicate its addresses. Furthermore two channels
// must be available, one for receiving and one for sending Messages.
//
// On closing down, an ApplicationAgent MUST close its MessageSender channel and MUST leave the MessageReceiver
// open. The supervising code MUST close the MessageReceiver of its subjects.
type ApplicationAgent interface {
	// Endpoints returns the EndpointIDs that this ApplicationAgent answers to.
	Endpoints() []bpv7.EndpointID

	// MessageReceiver is a channel on which the ApplicationAgent must listen for incoming Messages.
	MessageReceiver() chan Message

	// MessageSender is a channel to which the ApplicationAgent can send outgoing Messages.
	MessageSender() chan Message
}

// bagContainsEndpoint checks if some bag/array/slice of endpoints contains another collection of endpoints.
func bagContainsEndpoint(bag []bpv7.EndpointID, eids []bpv7.EndpointID) bool {
	matches := map[bpv7.EndpointID]struct{}{}

	for _, eid := range eids {
		matches[eid] = struct{}{}
	}

	for _, eid := range bag {
		if _, ok := matches[eid]; ok {
			return true
		}
	}
	return false
}

// bagHasEndpoint checks if some bag/array/slice of endpoints contains another endpoint.
func bagHasEndpoint(bag []bpv7.EndpointID, eid bpv7.EndpointID) bool {
	return bagContainsEndpoint(bag, []bpv7.EndpointID{eid})
}

// AppAgentContainsEndpoint checks if an ApplicationAgent listens to at least one of the requested endpoints.
func AppAgentContainsEndpoint(app ApplicationAgent, eids []bpv7.EndpointID) bool {
	return bagContainsEndpoint(app.Endpoints(), eids)
}

// AppAgentHasEndpoint checks if an ApplicationAgent listens to this endpoint.
func AppAgentHasEndpoint(app ApplicationAgent, eid bpv7.EndpointID) bool {
	return AppAgentContainsEndpoint(app, []bpv7.EndpointID{eid})
}
