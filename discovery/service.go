package discovery

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/geistesk/dtn7/cla/mtcp"
	"github.com/geistesk/dtn7/core"
	"github.com/schollz/peerdiscovery"
)

// DiscoveryService is a type to publish the node's CLAs to its network while
// discovering new peers. Internally UDP mulitcast packets are used.
type DiscoveryService struct {
	c *core.Core

	stopChan4 chan struct{}
	stopChan6 chan struct{}
}

func (ds *DiscoveryService) notify6(discovered peerdiscovery.Discovered) {
	discovered.Address = fmt.Sprintf("[%s]", discovered.Address)

	ds.notify(discovered)
}

func (ds *DiscoveryService) notify(discovered peerdiscovery.Discovered) {
	dms, err := NewDiscoveryMessagesFromCbor(discovered.Payload)
	if err != nil {
		log.WithFields(log.Fields{
			"discovery": ds,
			"peer":      discovered.Address,
			"error":     err,
		}).Warn("Peer discovery failed to parse incoming package")

		return
	}

	for _, dm := range dms {
		go ds.handleDiscovery(dm, discovered.Address)
	}
}

func (ds *DiscoveryService) handleDiscovery(dm DiscoveryMessage, addr string) {
	log.WithFields(log.Fields{
		"discovery": ds,
		"peer":      addr,
		"message":   dm,
	}).Debug("Peer discovery received a message")

	if dm.Type != MTCP {
		log.WithFields(log.Fields{
			"discovery": ds,
			"peer":      addr,
			"type":      uint(dm.Type),
		}).Warn("DiscoveryMessage's Type is unknown or unsupported")
		return
	}

	client := mtcp.NewMTCPClient(
		fmt.Sprintf("%s:%d", addr, dm.Port), dm.Endpoint, false)
	ds.c.RegisterConvergence(client)
}

// Close shuts the DiscoveryService down.
func (ds *DiscoveryService) Close() {
	if ds.stopChan4 != nil {
		ds.stopChan4 <- struct{}{}
	}

	if ds.stopChan6 != nil {
		ds.stopChan6 <- struct{}{}
	}
}

// NewDiscoveryService starts a new DiscoveryService and promotes the given
// DiscoveryMessages through IPv4 and/or IPv6, as specified in the parameters.
// Furthermore, received DiscoveryMessages will be processed.
func NewDiscoveryService(dms []DiscoveryMessage, c *core.Core, ipv4, ipv6 bool) (*DiscoveryService, error) {
	log.WithFields(log.Fields{
		"ipv4":    ipv4,
		"ipv6":    ipv6,
		"message": dms,
	}).Info("Started DiscoveryService")

	var ds = &DiscoveryService{
		c: c,
	}

	if ipv4 {
		ds.stopChan4 = make(chan struct{})
	}

	if ipv6 {
		ds.stopChan6 = make(chan struct{})
	}

	msg, err := DiscoveryMessagesToCbor(dms)
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
		{ipv4, DiscoveryAddress4, ds.stopChan4, peerdiscovery.IPv4, ds.notify},
		{ipv6, DiscoveryAddress6, ds.stopChan6, peerdiscovery.IPv6, ds.notify6},
	}

	for _, set := range sets {
		if !set.active {
			continue
		}

		set := peerdiscovery.Settings{
			Limit:            -1,
			Port:             fmt.Sprintf("%d", DiscoveryPort),
			MulticastAddress: set.multicastAddress,
			Payload:          msg,
			Delay:            10 * time.Second,
			TimeLimit:        -1,
			StopChan:         set.stopChan,
			AllowSelf:        true,
			IPVersion:        set.ipVersion,
			Notify:           set.notify,
		}

		go peerdiscovery.Discover(set)
	}

	return ds, nil
}
