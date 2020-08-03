// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/agent"
	"github.com/dtn7/dtn7-go/bundle"
)

// AgentManager is a proxy to connect different ApplicationAgents with the core package.
type AgentManager struct {
	core *Core

	mux *agent.MuxAgent

	closeSyn chan struct{}
	closeAck chan struct{}
}

// NewAgentManager creates a new AgentManager to proxy different ApplicationAgents within the core package.
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
func (manager *AgentManager) HasEndpoint(eid bundle.EndpointID) bool {
	return agent.AppAgentHasEndpoint(manager.mux, eid)
}

// Deliver an outgoing Bundle to a registered ApplicationAgent, addressed by the Bundle's destination.
func (manager *AgentManager) Deliver(bp BundlePack) error {
	b, bErr := bp.Bundle()
	if bErr != nil {
		return bErr
	}

	if !manager.HasEndpoint(b.PrimaryBlock.Destination) {
		log.WithField("bundle", b).Warn("AgentManager has no registered Agent for this outgoing Bundle")
		return fmt.Errorf("no registered ApplicationAgent for this Bundle's destination")
	}

	bp.RemoveConstraint(LocalEndpoint)
	if err := bp.Sync(); err != nil {
		log.WithField("bundle", b).WithError(err).Warn("AgentManager errored while sync'ing BundlePack")
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
