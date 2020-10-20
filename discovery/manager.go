// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
// SPDX-FileCopyrightText: 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package discovery

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/schollz/peerdiscovery"

	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/mtcp"
	"github.com/dtn7/dtn7-go/cla/tcpclv4"
	"github.com/dtn7/dtn7-go/routing"
)

// Manager publishes and receives Announcements.
type Manager struct {
	c *routing.Core

	stopChan4 chan struct{}
	stopChan6 chan struct{}
}

func (manager *Manager) notify6(discovered peerdiscovery.Discovered) {
	discovered.Address = fmt.Sprintf("[%s]", discovered.Address)

	manager.notify(discovered)
}

func (manager *Manager) notify(discovered peerdiscovery.Discovered) {
	announcements, err := UnmarshalAnnouncements(discovered.Payload)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"discovery": manager,
			"peer":      discovered.Address,
		}).Warn("Peer discovery failed to parse incoming package")

		return
	}

	for _, announcement := range announcements {
		go manager.handleDiscovery(announcement, discovered.Address)
	}
}

func (manager *Manager) handleDiscovery(announcement Announcement, addr string) {
	log.WithFields(log.Fields{
		"discovery": manager,
		"peer":      addr,
		"message":   announcement,
	}).Debug("Peer discovery received a message")

	if manager.c.HasEndpoint(announcement.Endpoint) {
		return
	}

	switch announcement.Type {
	case cla.MTCP:
		client := mtcp.NewMTCPClient(fmt.Sprintf("%s:%d", addr, announcement.Port), announcement.Endpoint, false)
		manager.c.RegisterConvergable(client)

	case cla.TCPCLv4:
		client := tcpclv4.DialTCP(fmt.Sprintf("%s:%d", addr, announcement.Port), manager.c.NodeId, false)
		manager.c.RegisterConvergable(client)

	default:
		log.WithFields(log.Fields{
			"discovery": manager,
			"peer":      addr,
			"type":      announcement.Type,
			"type-no":   uint(announcement.Type),
		}).Warn("Announcement's Type is unknown or unsupported")
		return
	}
}

// Close this Manager.
func (manager *Manager) Close() {
	for _, c := range []chan struct{}{manager.stopChan4, manager.stopChan6} {
		if c != nil {
			c <- struct{}{}
		}
	}
}

// NewManager for Announcements will be created and started.
func NewManager(announcements []Announcement, c *routing.Core, intervalSec uint, ipv4, ipv6 bool) (*Manager, error) {
	log.WithFields(log.Fields{
		"interval": intervalSec,
		"ipv4":     ipv4,
		"ipv6":     ipv6,
		"message":  announcements,
	}).Info("Started Manager")

	var manager = &Manager{c: c}
	if ipv4 {
		manager.stopChan4 = make(chan struct{})
	}
	if ipv6 {
		manager.stopChan6 = make(chan struct{})
	}

	msg, err := MarshalAnnouncements(announcements)
	if err != nil {
		return nil, err
	}

	sets := []struct {
		active           bool
		multicastAddress string
		stopChan         chan struct{}
		ipVersion        peerdiscovery.IPVersion
		notify           func(discovered peerdiscovery.Discovered)
	}{
		{ipv4, address4, manager.stopChan4, peerdiscovery.IPv4, manager.notify},
		{ipv6, address6, manager.stopChan6, peerdiscovery.IPv6, manager.notify6},
	}

	for _, set := range sets {
		if !set.active {
			continue
		}

		set := peerdiscovery.Settings{
			Limit:            -1,
			Port:             fmt.Sprintf("%d", port),
			MulticastAddress: set.multicastAddress,
			Payload:          msg,
			Delay:            time.Duration(intervalSec) * time.Second,
			TimeLimit:        -1,
			StopChan:         set.stopChan,
			AllowSelf:        true,
			IPVersion:        set.ipVersion,
			Notify:           set.notify,
		}

		discoverErrChan := make(chan error)
		go func() {
			_, discoverErr := peerdiscovery.Discover(set)
			discoverErrChan <- discoverErr
		}()

		select {
		case discoverErr := <-discoverErrChan:
			if discoverErr != nil {
				return nil, discoverErr
			}

		case <-time.After(time.Second):
			break
		}
	}

	return manager, nil
}
