// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/agent"
	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// AgentManager is a proxy to connect different ApplicationAgents with the routing package.
type AgentManager struct {
	core *Core

	mux *agent.MuxAgent

	closeSyn chan struct{}
	closeAck chan struct{}
}

// NewAgentManager creates a new AgentManager to proxy different ApplicationAgents within the routing package.
func NewAgentManager(core *Core) (manager *AgentManager) {
	manager = &AgentManager{
		core:     core,
		mux:      agent.NewMuxAgent(),
		closeSyn: make(chan struct{}),
		closeAck: make(chan struct{}),
	}

	go manager.handler()

	return
}

func (manager *AgentManager) handler() {
	defer func() {
		close(manager.closeAck)
	}()

	for {
		select {
		case <-manager.closeSyn:
			return

		case msg := <-manager.mux.MessageSender():
			manager.handleMessage(msg)
		}
	}
}

func (manager *AgentManager) handleMessage(msg agent.Message) {
	switch msg := msg.(type) {
	case agent.BundleMessage:
		log.WithField("bundle", msg.Bundle).Debug("AgentManager received Bundle from client")
		manager.core.SendBundle(&msg.Bundle)

	// TODO
	//case agent.SyscallRequestMessage:
	//case agent.ShutdownMessage:

	default:
		log.WithField("message", msg).Warn("AgentManager received unsupported message")
	}
}

// Register a new ApplicationAgent.
func (manager *AgentManager) Register(appAgent agent.ApplicationAgent) {
	manager.mux.Register(appAgent)
}

// HasEndpoint checks if some specific EndpointID is registered for some ApplicationAgent.
func (manager *AgentManager) HasEndpoint(eid bpv7.EndpointID) bool {
	return agent.AppAgentHasEndpoint(manager.mux, eid)
}

// Deliver a Bundle to a registered ApplicationAgent, addressed by the Bundle's destination.
func (manager *AgentManager) Deliver(descriptor BundleDescriptor) error {
	b, bErr := descriptor.Bundle()
	if bErr != nil {
		return bErr
	}

	if !manager.HasEndpoint(b.PrimaryBlock.Destination) {
		log.WithField("bundle", b).Warn("AgentManager has no registered Agent for this Bundle")
		return fmt.Errorf("no registered ApplicationAgent for this Bundle's destination")
	}

	descriptor.RemoveConstraint(LocalEndpoint)
	if err := descriptor.Sync(); err != nil {
		log.WithField("bundle", b).WithError(err).Warn("AgentManager errored while synchronizing BundleDescriptor")
		return err
	}

	log.WithField("bundle", b).Debug("AgentManager delivers Bundle to client")
	manager.mux.MessageReceiver() <- agent.BundleMessage{Bundle: *b}
	return nil
}

// Close down this AgentManager and its underlying ApplicationAgents.
func (manager *AgentManager) Close() error {
	manager.mux.MessageReceiver() <- agent.ShutdownMessage{}

	close(manager.closeSyn)
	select {
	case <-manager.closeAck:
		return nil

	case <-time.After(time.Second):
		return fmt.Errorf("closing timed out after a second")
	}
}
