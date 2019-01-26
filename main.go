package main

import (
	"fmt"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla/stcp"
)

func setupServer() {
	serv := stcp.NewSTCPServer(":9000", bundle.MustNewEndpointID("dtn", "gumo"))

	go func() {
		chnl := serv.Channel()

		for {
			select {
			case bndl := <-chnl:
				fmt.Println(bndl)

			case <-time.After(750 * time.Millisecond):
				serv.Close()
				return
			}
		}
	}()
}

func main() {
	setupServer()

	client, err := stcp.NewAnonymousSTCPClient("localhost:9000")
	if err != nil {
		panic(err)
	}

	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn", "dest"),
			bundle.MustNewEndpointID("dtn", "src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		panic(err)
	}

	if err = client.Send(bndl); err != nil {
		panic(err)
	}
	time.Sleep(300 * time.Millisecond)

	if err = client.Send(bndl); err != nil {
		panic(err)
	}
	time.Sleep(100 * time.Millisecond)

	if err = client.Send(bndl); err != nil {
		panic(err)
	}
	time.Sleep(200 * time.Millisecond)

	client.Close()

	time.Sleep(time.Second)
}
