// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"fmt"
)

// dummyHub connects multiple dummyModems and helps mocking the bbc package.
type dummyHub struct {
	modems []*dummyModem

	fragmentCounter int
	fragmentDrop    int
}

// newDummyHub creates a new dummyHub.
func newDummyHub() *dummyHub {
	return &dummyHub{}
}

// newDummyHubDrop creates a new dummyHub which drops each nth Fragment.
func newDummyHubDrop(n int) *dummyHub {
	return &dummyHub{fragmentDrop: n}
}

// connect a dummyModem to this dummyHub. This method is called from the newDummyModem function.
func (dh *dummyHub) connect(m *dummyModem) {
	dh.modems = append(dh.modems, m)
}

// receive a Fragment to this dummyHub and distribute it to its dummyModems.
func (dh *dummyHub) receive(f Fragment) {
	dh.fragmentCounter++
	if dh.fragmentDrop != 0 && dh.fragmentCounter%dh.fragmentDrop == 0 {
		return
	}

	for _, m := range dh.modems {
		m.deliver(f)
	}
}

// dummyModem is a mocking Modem used for testing.
type dummyModem struct {
	mtu    int
	hub    *dummyHub
	inChan chan Fragment

	closedSyn chan struct{}
	closedAck chan struct{}
}

// newDummyModem creates a new dummyModem and connects itself to a dummyHub.
func newDummyModem(mtu int, hub *dummyHub) *dummyModem {
	d := &dummyModem{
		mtu:    mtu,
		hub:    hub,
		inChan: make(chan Fragment, 10),

		closedSyn: make(chan struct{}),
		closedAck: make(chan struct{}),
	}
	hub.connect(d)

	return d
}

// deliver a Fragment from a dummyHub to this dummyModem.
func (d *dummyModem) deliver(f Fragment) {
	d.inChan <- f
}

func (d *dummyModem) Mtu() int {
	return d.mtu
}

func (d *dummyModem) Send(f Fragment) error {
	d.hub.receive(f)
	return nil
}

func (d *dummyModem) Receive() (f Fragment, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("receive recovered: %v, previous error: %v", r, err)
		}
	}()

	select {
	case f = <-d.inChan:
		// return received Fragment
	case <-d.closedSyn:
		close(d.closedAck)
	}

	return
}

func (d *dummyModem) Close() error {
	close(d.closedSyn)
	<-d.closedAck

	return nil
}

func (d *dummyModem) String() string {
	return fmt.Sprintf("dummymodem/mtu:%d", d.mtu)
}
