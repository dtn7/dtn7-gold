// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpclv4

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func TestSimpleWebSocketNetwork(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	manager := cla.NewManager()
	listener := ListenWebSocket(bundle.MustNewEndpointID("dtn://server/"))
	manager.Register(listener)

	recvBundle := make(chan bundle.Bundle)
	go func() {
		for msg := range manager.Channel() {
			if msg.MessageType == cla.ReceivedBundle {
				recvBundle <- *msg.Message.(cla.ConvergenceReceivedBundle).Bundle
			}
		}
	}()

	addr := fmt.Sprintf("localhost:%d", randomPort(t))

	httpMux := http.NewServeMux()
	httpMux.Handle("/tcpclv4", listener)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: httpMux,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	time.Sleep(100 * time.Millisecond)

	client := DialWebSocket("ws://"+addr+"/tcpclv4", bundle.MustNewEndpointID("dtn://client/"), false)
	if err, _ := client.Start(); err != nil {
		t.Fatal(err)
	}

	go func() {
		for range client.Channel() {
		}
	}()

	b, bErr := bundle.Builder().
		CRC(bundle.CRC32).
		Source("dtn://client/").
		Destination("dtn://server/").
		CreationTimestampNow().
		Lifetime(30 * time.Minute).
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	} else if err := client.Send(b); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b, <-recvBundle) {
		t.Fatalf("Bundles differ: %v, %v", b, recvBundle)
	}
}
