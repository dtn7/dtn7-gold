// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"time"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// dummyStage is used for internal testing.
//
// The delay parameter specifies the time this Stage "takes" resp. blocks.
type dummyStage struct {
	delay time.Duration

	state *State

	closeChan chan struct{}
	finChan   chan struct{}
}

func (ds *dummyStage) Start(state *State) {
	ds.state = state

	ds.closeChan = make(chan struct{})
	ds.finChan = make(chan struct{})

	go ds.handler()
}

func (ds *dummyStage) handler() {
	select {
	case <-ds.closeChan:
		ds.state.StageError = StageClose
	case <-time.After(ds.delay):
	}

	close(ds.finChan)
}

func (ds *dummyStage) Exchanges() (outgoing chan<- msgs.Message, incoming <-chan msgs.Message, exchangeOk bool) {
	return nil, nil, false
}

func (ds *dummyStage) Close() error {
	close(ds.closeChan)
	return nil
}

func (ds *dummyStage) Finished() <-chan struct{} {
	return ds.finChan
}
