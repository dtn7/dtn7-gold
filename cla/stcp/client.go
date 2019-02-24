package stcp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// STCPClient is an implementation of a Simple TCP Convergence-Layer client
// which connects to a STCP server to send bundles.
type STCPClient struct {
	conn  net.Conn
	peer  bundle.EndpointID
	mutex sync.Mutex

	address string
}

// NewSTCPClient creates a new STCPClient, connected to the given address for
// the registered endpoint ID.
func NewSTCPClient(address string, peer bundle.EndpointID) *STCPClient {
	return &STCPClient{
		peer:    peer,
		address: address,
	}
}

// NewSTCPClient creates a new STCPClient, connected to the given address.
func NewAnonymousSTCPClient(address string) *STCPClient {
	return NewSTCPClient(address, bundle.DtnNone())
}

// Start starts this STCPClient and might return an error and a boolean
// indicating if another Start should be tried later.
func (client *STCPClient) Start() (error, bool) {
	conn, err := net.DialTimeout("tcp", client.address, time.Second)
	if err == nil {
		client.conn = conn
	}

	return err, true
}

// Send transmits a bundle to this STCPClient's endpoint.
func (client *STCPClient) Send(bndl bundle.Bundle) (err error) {
	client.mutex.Lock()

	var enc = codec.NewEncoder(client.conn, new(codec.CborHandle))
	err = enc.Encode(newDataUnit(bndl))

	client.mutex.Unlock()

	return
}

// Close closes the STCPClient's connection.
func (client *STCPClient) Close() {
	client.mutex.Lock()
	client.conn.Close()
	client.mutex.Unlock()
}

// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
// if it's known. Otherwise the zero endpoint will be returned.
func (client *STCPClient) GetPeerEndpointID() bundle.EndpointID {
	return client.peer
}

// Address should return a unique address string to both identify this
// ConvergenceSender and ensure it will not opened twice.
func (client *STCPClient) Address() string {
	return client.address
}

func (client *STCPClient) String() string {
	if client.conn != nil {
		return fmt.Sprintf("stcp://%v", client.conn.RemoteAddr())
	} else {
		return fmt.Sprintf("stcp://%s", client.address)
	}
}
