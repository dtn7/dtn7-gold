package bbc

import (
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
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

// The following test relies on a system which is equipped with two rf95modems..
/*
func TestLoRaConnector(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	m0, _ := NewRf95Modem("/dev/ttyUSB0")
	c0 := NewConnector(m0, true)
	_, _ = c0.Start()

	m1, _ := NewRf95Modem("/dev/ttyUSB1")
	c1 := NewConnector(m1, true)
	_, _ = c1.Start()

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

	if err := c0.Send(&b); err != nil {
		t.Fatal(err)
	}

	uff := <-c1.Channel()
	log.WithField("message", uff).Info("Received Bundle, yay")

	c0.Close()
	c1.Close()
}
*/
