package discovery

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/schollz/peerdiscovery"
)

type DiscoveryService struct {
	settings peerdiscovery.Settings
}

func (ds *DiscoveryService) notify(discovered peerdiscovery.Discovered) {
	dm, err := NewDiscoveryMessageFromCbor(discovered.Payload)
	if err != nil {
		return
	}

	log.Printf("Peer discovery discovered %v at %v", dm, discovered.Address)
}

func (ds *DiscoveryService) Close() {
	ds.settings.StopChan <- struct{}{}
}

func NewDiscoveryService(dm DiscoveryMessage) (*DiscoveryService, error) {
	var ds = &DiscoveryService{}

	msg, err := dm.Cbor()
	if err != nil {
		return nil, err
	}

	set := peerdiscovery.Settings{
		Limit:            -1,
		Port:             fmt.Sprintf("%d", DiscoveryPort),
		MulticastAddress: DiscoveryAddress4,
		Payload:          msg,
		Delay:            time.Duration(rand.Int31n(5)+5) * time.Second,
		TimeLimit:        -1,
		StopChan:         make(chan struct{}),
		AllowSelf:        true,
		IPVersion:        peerdiscovery.IPv4, // TODO
		Notify:           ds.notify,
	}
	ds.settings = set

	go peerdiscovery.Discover(ds.settings)

	return ds, nil
}
