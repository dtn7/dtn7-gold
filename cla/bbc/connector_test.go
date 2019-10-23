package bbc

import (
	"github.com/dtn7/dtn7-go/bundle"
	"testing"
)

func TestConnector(t *testing.T) {
	hub := newDummyHub()
	c := NewConnector(newDummyModem(23, hub), true)
	_, _ = c.Start()

	b, bErr := bundle.Builder().
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime("10m").
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	if err := c.Send(&b); err != nil {
		t.Fatal(err)
	}

	uff := <-c.Channel()
	t.Log(uff)
}
