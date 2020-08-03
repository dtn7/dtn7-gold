// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import "github.com/dtn7/dtn7-go/bundle"

// ApplicationAgent is an interface to describe application agents, which can both receive and transmit Bundles.
// Each implementation must provide the following methods to communicate its addresses. Furthermore two channels
// must be available, one for receiving and one for sending Messages.
//
// On closing down, an ApplicationAgent MUST close its MessageSender channel and MUST leave the MessageReceiver
// open. The supervising code MUST close the MessageReceiver of its subjects.
type ApplicationAgent interface {
	// Endpoints returns the EndpointIDs that this ApplicationAgent answers to.
	Endpoints() []bundle.EndpointID

	// MessageReceiver is a channel on which the ApplicationAgent must listen for incoming Messages.
	MessageReceiver() chan Message

	// MessageSender is a channel to which the ApplicationAgent can send outgoing Messages.
	MessageSender() chan Message
}

// bagContainsEndpoint checks if some bag/array/slice of endpoints contains another collection of endpoints.
func bagContainsEndpoint(bag []bundle.EndpointID, eids []bundle.EndpointID) bool {
	matches := map[bundle.EndpointID]struct{}{}

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
func bagHasEndpoint(bag []bundle.EndpointID, eid bundle.EndpointID) bool {
	return bagContainsEndpoint(bag, []bundle.EndpointID{eid})
}

// AppAgentContainsEndpoint checks if an ApplicationAgent listens to at least one of the requested endpoints.
func AppAgentContainsEndpoint(app ApplicationAgent, eids []bundle.EndpointID) bool {
	return bagContainsEndpoint(app.Endpoints(), eids)
}

// AppAgentHasEndpoint checks if an ApplicationAgent listens to this endpoint.
func AppAgentHasEndpoint(app ApplicationAgent, eid bundle.EndpointID) bool {
	return AppAgentContainsEndpoint(app, []bundle.EndpointID{eid})
}
