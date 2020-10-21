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

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/cla/mtcp"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4"
)

// Manager publishes and receives Announcements.
type Manager struct {
	NodeId       bpv7.EndpointID
	RegisterFunc func(cla.Convergable)

	stopChan4 chan struct{}
	stopChan6 chan struct{}
}

// NewManager for Announcements will be created and started.
func NewManager(
	nodeId bpv7.EndpointID, registerFunc func(cla.Convergable),
	announcements []Announcement, announcementInterval time.Duration,
	ipv4, ipv6 bool) (*Manager, error) {

	var manager = &Manager{
		NodeId:       nodeId,
		RegisterFunc: registerFunc,
	}
	if ipv4 {
		manager.stopChan4 = make(chan struct{})
	}
	if ipv6 {
		manager.stopChan6 = make(chan struct{})
	}

	log.WithFields(log.Fields{
		"interval":      announcementInterval,
		"IPv4":          ipv4,
		"IPv6":          ipv6,
		"announcements": announcements,
	}).Info("Starting Manager")

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
			Delay:            announcementInterval,
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

	if manager.NodeId.SameNode(announcement.Endpoint) {
		return
	}

	var convergable cla.Convergable
	switch announcement.Type {
	case cla.MTCP:
		convergable = mtcp.NewMTCPClient(fmt.Sprintf("%s:%d", addr, announcement.Port), announcement.Endpoint, false)

	case cla.TCPCLv4:
		convergable = tcpclv4.DialTCP(fmt.Sprintf("%s:%d", addr, announcement.Port), manager.NodeId, false)

	default:
		log.WithFields(log.Fields{
			"discovery": manager,
			"peer":      addr,
			"type":      announcement.Type,
			"type-no":   uint(announcement.Type),
		}).Warn("Announcement's Type is unknown or unsupported")
		return
	}

	manager.RegisterFunc(convergable)
}

// Close this Manager.
func (manager *Manager) Close() {
	for _, c := range []chan struct{}{manager.stopChan4, manager.stopChan6} {
		if c != nil {
			c <- struct{}{}
		}
	}
}
