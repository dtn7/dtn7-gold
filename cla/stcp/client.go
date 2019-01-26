package stcp

import (
	"net"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// STCPClient is an implementation of a Simple TCP Convergence-Layer client
// which connects to a STCP server to send bundles.
type STCPClient struct {
	conn net.Conn
	peer bundle.EndpointID
}

// NewSTCPClient creates a new STCPClient, connected to the given address for
// the registered endpoint ID.
func NewSTCPClient(address string, peer bundle.EndpointID) (client *STCPClient, err error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}

	return &STCPClient{conn, peer}, nil
}

// NewSTCPClient creates a new STCPClient, connected to the given address.
func NewAnonymousSTCPClient(address string) (client *STCPClient, err error) {
	return NewSTCPClient(address, bundle.DtnNone())
}

// Send transmits a bundle to this STCPClient's endpoint.
func (client *STCPClient) Send(bndl bundle.Bundle) error {
	var enc = codec.NewEncoder(client.conn, new(codec.CborHandle))
	return enc.Encode(newDataUnit(bndl))
}

// Close closes the STCPClient's connection.
func (client *STCPClient) Close() error {
	return client.conn.Close()
}

// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
// if it's known. Otherwise the zero endpoint will be returned.
func (client STCPClient) GetPeerEndpointID() bundle.EndpointID {
	return client.peer
}
