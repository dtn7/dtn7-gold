package main

import (
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla/stcp"
)

func main() {
	go stcp.LaunchReceiver(9000)

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

	time.Sleep(time.Second)
}
