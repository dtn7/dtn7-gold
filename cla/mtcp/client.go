package mtcp

import (
	"bufio"
	"fmt"
	"net"

	"github.com/dtn7/dtn7/bundle"
)

// MTCPClient is an implementation of a Minimal TCP Convergence-Layer client
// which connects to a MTCP server to send bundles.
type MTCPClient struct {
	address   string
	peer      bundle.EndpointID
	permanent bool
}

// NewMTCPClient creates a new MTCPClient, connected to the given address for
// the registered endpoint ID. The permanent flag indicates if this MTCPClient
// should never be removed from the core.
func NewMTCPClient(address string, peer bundle.EndpointID, permanent bool) *MTCPClient {
	return &MTCPClient{
		address:   address,
		peer:      peer,
		permanent: permanent,
	}
}

// NewMTCPClient creates a new MTCPClient, connected to the given address. The
// permanent flag indicates if this MTCPClient should never be removed from
// the core.
func NewAnonymousMTCPClient(address string, permanent bool) *MTCPClient {
	return NewMTCPClient(address, bundle.DtnNone(), permanent)
}

// connect establishes a connection.
func (client *MTCPClient) connect() (net.Conn, error) {
	return net.Dial("tcp", client.address)
}

// Start starts this MTCPClient and might return an error and a boolean
// indicating if another Start should be tried later.
func (client *MTCPClient) Start() (error, bool) {
	conn, err := client.connect()
	if err == nil {
		conn.Close()
	}

	return err, true
}

// Send transmits a bundle to this MTCPClient's endpoint.
func (client *MTCPClient) Send(bndl *bundle.Bundle) (err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("MTCPClient.Send: %v", r)
		}
	}()

	conn, connErr := client.connect()
	if connErr != nil {
		err = connErr
		return
	}
	defer conn.Close()

	connWriter := bufio.NewWriter(conn)
	defer connWriter.Flush()

	bndlData := bndl.ToCbor()

	// CBOR byte string with defined length
	if bndlLen := len(bndlData); bndlLen >= 1<<16 {
		_, err = connWriter.Write([]byte{
			0x5A,
			byte((bndlLen >> 24) & 0xFF),
			byte((bndlLen >> 16) & 0xFF),
			byte((bndlLen >> 8) & 0xFF),
			byte(bndlLen & 0xFF),
		})
	} else if bndlLen >= 1<<8 {
		_, err = connWriter.Write([]byte{
			0x59,
			byte((bndlLen >> 8) & 0xFF),
			byte(bndlLen & 0xFF),
		})
	} else {
		_, err = connWriter.Write([]byte{
			0x58,
			byte(bndlLen & 0xFF),
		})
	}
	if err != nil {
		return
	}

	if _, err = connWriter.Write(bndlData); err != nil {
		return
	}

	return
}

// Close closes the MTCPClient's connection.
func (_ *MTCPClient) Close() {}

// GetPeerEndpointID returns the endpoint ID assigned to this CLA's peer,
// if it's known. Otherwise the zero endpoint will be returned.
func (client *MTCPClient) GetPeerEndpointID() bundle.EndpointID {
	return client.peer
}

// Address should return a unique address string to both identify this
// ConvergenceSender and ensure it will not opened twice.
func (client *MTCPClient) Address() string {
	return client.address
}

// IsPermanent returns true, if this CLA should not be removed after failures.
func (client *MTCPClient) IsPermanent() bool {
	return client.permanent
}

func (client *MTCPClient) String() string {
	return fmt.Sprintf("mtcp://%s", client.address)
}
