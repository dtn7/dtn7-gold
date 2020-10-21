// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cla

import (
	"fmt"
	"time"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// mockConvRec mocks a ConvergenceReceiver where all fields are directly editable.
type mockConvRec struct {
	// startable and startableRetry defines if this mockConvRec can be started.
	startable      bool
	startableRetry bool

	// reportChan is the channel, which can be directly used for mocking purpose.
	reportChan chan ConvergenceStatus

	// permanent defines if this mockConvRec is handled as permanent.
	permanent bool

	// address is the unique address and endpointId this mockConvRec's Endpoint ID.
	address    string
	endpointId bpv7.EndpointID
}

func newMockConvRec(startable bool, address string, eid bpv7.EndpointID) *mockConvRec {
	return &mockConvRec{
		startable:      startable,
		startableRetry: true,
		reportChan:     make(chan ConvergenceStatus),
		permanent:      false,
		address:        address,
		endpointId:     eid,
	}
}

func (m *mockConvRec) Start() (err error, retry bool) {
	if !m.startable {
		err = fmt.Errorf("startable := false")
	}

	retry = m.startableRetry
	return
}

func (_ *mockConvRec) Close() error {
	return nil
}

func (m *mockConvRec) Channel() chan ConvergenceStatus { return m.reportChan }

func (m *mockConvRec) Address() string { return m.address }

func (m *mockConvRec) IsPermanent() bool { return m.permanent }

func (m *mockConvRec) GetEndpointID() bpv7.EndpointID { return m.endpointId }

// mockConvSender mocks a ConvergenceSender where all fields are directly editable.
type mockConvSender struct {
	// startable and startableRetry defines if this mockConvRec can be started.
	startable      bool
	startableRetry bool

	// reportChan is the channel, which can be directly used for mocking purpose.
	reportChan chan ConvergenceStatus

	// permanent defines if this mockConvRec is handled as permanent.
	permanent bool

	// address is the unique address and peerEndpointId the peer's Endpoint ID.
	address        string
	peerEndpointId bpv7.EndpointID

	// sentBndls is an array of all sent bundles, sendFail indicates if sending should fail.
	sentBndls []bpv7.Bundle
	sendFail  bool
}

func newMockConvSender(startable bool, address string, eid bpv7.EndpointID) *mockConvSender {
	return &mockConvSender{
		startable:      startable,
		startableRetry: true,
		reportChan:     make(chan ConvergenceStatus),
		permanent:      false,
		address:        address,
		peerEndpointId: eid,
		sentBndls:      make([]bpv7.Bundle, 0),
		sendFail:       false,
	}
}

func (m *mockConvSender) Start() (err error, retry bool) {
	if !m.startable {
		err = fmt.Errorf("startable := false")
	}

	retry = m.startableRetry

	go func(m *mockConvSender) {
		time.Sleep(10 * time.Millisecond)
		m.reportChan <- NewConvergencePeerAppeared(m, m.GetPeerEndpointID())
	}(m)

	return
}

func (_ *mockConvSender) Close() error {
	return nil
}

func (m *mockConvSender) Channel() chan ConvergenceStatus { return m.reportChan }

func (m *mockConvSender) Address() string { return m.address }

func (m *mockConvSender) IsPermanent() bool { return m.permanent }

func (m *mockConvSender) GetPeerEndpointID() bpv7.EndpointID { return m.peerEndpointId }

func (m *mockConvSender) Send(bndl bpv7.Bundle) error {
	if m.sendFail {
		return fmt.Errorf("sendFail := true")
	}

	m.sentBndls = append(m.sentBndls, bndl)
	return nil
}
