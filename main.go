package main

import (
	"fmt"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla/stcp"
)

func setupServer() {
	reportChan := make(chan bundle.Bundle)

	serv := stcp.NewSTCPServer(":9000", reportChan)
	serv.Construct()

	go func() {
		for {
			select {
			case bndl := <-reportChan:
				fmt.Println(bndl)

			case <-time.After(750 * time.Millisecond):
				serv.Destruct()
				return
			}
		}
	}()
}

func main() {
	setupServer()

	var bndl, err = bundle.NewBundle(
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

	stcp.SendPoC("localhost:9000", bndl)
	time.Sleep(300 * time.Millisecond)
	stcp.SendPoC("localhost:9000", bndl)
	time.Sleep(150 * time.Millisecond)
	stcp.SendPoC("localhost:9000", bndl)

	time.Sleep(time.Second)
}
