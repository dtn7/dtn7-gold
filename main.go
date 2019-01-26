package main

import (
	"fmt"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
	"github.com/geistesk/dtn7/cla/stcp"
	"github.com/geistesk/dtn7/core"
)

func setupServer(port int) cla.ConvergenceReceiver {
	var serv = stcp.NewSTCPServer(
		fmt.Sprintf(":%d", port),
		bundle.MustNewEndpointID("ipn", fmt.Sprintf("23.%d", port)))

	go func() {
		chnl := serv.Channel()

		for {
			select {
			case bndl := <-chnl:
				fmt.Printf("Server %d: %v\n", port, bndl)
			}
		}
	}()

	return serv
}

func setupCore() (aa *core.ApplicationAgent, pa *core.ProtocolAgent) {
	aa = new(core.ApplicationAgent)
	pa = new(core.ProtocolAgent)

	aa.ProtocolAgent = pa

	pa.ApplicationAgent = aa

	for i := 9001; i <= 9003; i++ {
		convClient, err := stcp.NewSTCPClient(
			fmt.Sprintf("localhost:%d", i),
			bundle.MustNewEndpointID("ipn", fmt.Sprintf("23.%d", i)))
		if err != nil {
			panic(err)
		}

		pa.ConvergenceSenders = append(pa.ConvergenceSenders, convClient)
	}

	return
}

func main() {
	servs := []cla.ConvergenceReceiver{
		setupServer(9001),
		setupServer(9002),
		setupServer(9003),
	}

	_, pa := setupCore()

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn", "dest"),
			bundle.DtnNone(),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		panic(err)
	}

	fmt.Println("Should be delivered to all endpoints, unknown endpoint")
	pa.Transmit(core.NewBundlePack(bndl))

	time.Sleep(time.Second)

	bndl.PrimaryBlock.Destination = bundle.MustNewEndpointID("ipn", "23.9001")
	fmt.Println("\n\nShould be delivered to 9001, specified endpoint")
	pa.Transmit(core.NewBundlePack(bndl))

	time.Sleep(time.Second)

	for _, serv := range servs {
		serv.Close()
	}
}
