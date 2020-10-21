// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"sync"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// MuxAgent mimics an ApplicationAgent to be used as a multiplexer for different ApplicationAgents.
type MuxAgent struct {
	sync.Mutex

	receiver chan Message
	sender   chan Message

	children []ApplicationAgent
}

// NewMuxAgent creates a new MuxAgent used to multiplex different ApplicationAgents.
func NewMuxAgent() (mux *MuxAgent) {
	mux = &MuxAgent{
		receiver: make(chan Message),
		sender:   make(chan Message),
	}

	go mux.handle()

	return
}

func (mux *MuxAgent) handle() {
	defer close(mux.sender)

	for msg := range mux.receiver {
		mux.Lock()
		for _, child := range mux.children {
			if rec := msg.Recipients(); rec == nil || AppAgentContainsEndpoint(child, rec) {
				child.MessageReceiver() <- msg
			}
		}
		mux.Unlock()

		if _, isShutdown := msg.(ShutdownMessage); isShutdown {
			return
		}
	}
}

// Register a new ApplicationAgent for this multiplexer.
// If this ApplicationAgent closes its channel or broadcasts a ShutdownMessage, it will be unregistered.
func (mux *MuxAgent) Register(agent ApplicationAgent) {
	mux.Lock()
	defer mux.Unlock()

	mux.children = append(mux.children, agent)
	go mux.handleChild(agent)
}

func (mux *MuxAgent) handleChild(agent ApplicationAgent) {
	for msg := range agent.MessageSender() {
		if _, isShutdown := msg.(ShutdownMessage); isShutdown {
			break
		}

		mux.sender <- msg
	}

	mux.unregister(agent)
}

// unregister a previously registered ApplicationAgent.
// This will also automatically shutdown this ApplicationAgent.
func (mux *MuxAgent) unregister(agent ApplicationAgent) {
	mux.Lock()
	defer mux.Unlock()

	close(agent.MessageReceiver())

	for i, child := range mux.children {
		if child == agent {
			mux.children = append(mux.children[:i], mux.children[i+1:]...)
			break
		}
	}
}

func (mux *MuxAgent) Endpoints() (endpoints []bpv7.EndpointID) {
	mux.Lock()
	defer mux.Unlock()

	for _, child := range mux.children {
		endpoints = append(endpoints, child.Endpoints()...)
	}
	return
}

func (mux *MuxAgent) MessageReceiver() chan Message {
	return mux.receiver
}

func (mux *MuxAgent) MessageSender() chan Message {
	return mux.sender
}
