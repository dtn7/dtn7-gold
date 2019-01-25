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
}

// NewSTCPClient creates a new STCPClient, connected to the given address.
func NewSTCPClient(address string) (client *STCPClient, err error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}

	return &STCPClient{conn}, nil
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
