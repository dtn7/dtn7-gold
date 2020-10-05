// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"time"
)

// dummyStage is used for internal testing.
//
// The delay parameter specifies the time this Stage "takes" resp. blocks.
type dummyStage struct {
	delay time.Duration

	state     *State
	closeChan <-chan struct{}
}

func (ds *dummyStage) Handle(state *State, closeChan <-chan struct{}) {
	ds.state = state
	ds.closeChan = closeChan

	select {
	case <-ds.closeChan:
		ds.state.StageError = StageClose
	case <-time.After(ds.delay):
	}
}
