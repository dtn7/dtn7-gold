package main

import (
	"fmt"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla/stcp"
	"github.com/geistesk/dtn7/core"
)

func createClient(port int, endpoint bundle.EndpointID) *core.Core {
	var c *core.Core = new(core.Core)

	c.Store = core.NoStore{}

	c.RegisterConvergenceReceiver(
		stcp.NewSTCPServer(fmt.Sprintf(":%d", port), endpoint))

	c.AppEndpoints = []bundle.EndpointID{
		bundle.MustNewEndpointID("dtn", fmt.Sprintf("resp%d", port))}

	return c
}

func connectClientTo(c *core.Core, dest string, destEndpoint bundle.EndpointID) {
	client, err := stcp.NewSTCPClient(dest, destEndpoint)
	if err != nil {
		panic(err)
	}

	c.RegisterConvergenceSender(client)
}

func main() {
	// Create three DTN clients
	ep1 := bundle.MustNewEndpointID("ipn", "23.9001")
	ep2 := bundle.MustNewEndpointID("ipn", "23.9002")
	ep3 := bundle.MustNewEndpointID("ipn", "23.9003")

	cl1 := createClient(9001, ep1)
	cl2 := createClient(9002, ep2)
	cl3 := createClient(9003, ep3)

	// Connect 1 <-> 2 and 2 <-> 3
	connectClientTo(cl1, "localhost:9002", ep2)
	connectClientTo(cl2, "localhost:9001", ep1)
	connectClientTo(cl2, "localhost:9003", ep3)
	connectClientTo(cl3, "localhost:9002", ep2)

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.StatusRequestReception|bundle.StatusRequestDelivery,
			ep3,
			ep1,
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 1000000),
		[]bundle.CanonicalBlock{
			bundle.NewHopCountBlock(1, 0, bundle.NewHopCount(23)),
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		panic(err)
	}

	cl1.Transmit(core.NewBundlePack(bndl))

	time.Sleep(time.Second)
}
