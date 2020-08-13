// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func getRandomPort(t *testing.T) (port int) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	port = l.Addr().(*net.TCPAddr).Port

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	return
}

func TestTcpSimple(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	serverAddr := fmt.Sprintf("localhost:%d", getRandomPort(t))
	listener := NewTcpListener(serverAddr, bundle.MustNewEndpointID("dtn://server/"))

	manager := cla.NewManager()
	manager.Register(listener)

	go func() {
		for status := range manager.Channel() {
			t.Logf("CLA Manager received: %v", status)
		}
	}()

	client := DialTcp(serverAddr, bundle.MustNewEndpointID("dtn://client/"), false)
	if err, retry := client.Start(); err != nil {
		t.Fatalf("Failed to start client: %v %v", err, retry)
	}

	go func() {
		for status := range client.Channel() {
			t.Logf("Client channel sent: %v", status)
		}
	}()

	time.Sleep(time.Second)

	t.Log("Closing down")
	client.Close()
	time.Sleep(250 * time.Millisecond)
}
