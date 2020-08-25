// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// StageHandler executes a sequence of Stages and passes the State from one Stage to another. Errors might be propagated
// back through the Error method.
type StageHandler struct {
	stages []Stage
	state  *State

	errChan   chan error
	closeChan chan struct{}
}

// NewStageHandler for a slice of Stages, Message channels and a Configuration.
func NewStageHandler(stages []Stage, msgIn, msgOut chan msgs.Message, config Configuration) (sh *StageHandler) {
	sh = &StageHandler{
		stages: stages,
		state: &State{
			Configuration: config,
			MsgIn:         msgIn,
			MsgOut:        msgOut,
			StageError:    nil,
		},

		errChan:   make(chan error),
		closeChan: make(chan struct{}),
	}

	go sh.handler()

	return
}

func (sh *StageHandler) handler() {
	defer close(sh.errChan)

	for i := 0; i < len(sh.stages); i++ {
		stage := sh.stages[i]
		stage.Start(sh.state)

		select {
		case <-sh.closeChan:
			_ = stage.Close()
			return

		case <-stage.Finished():
			if err := sh.state.StageError; err != nil {
				sh.errChan <- err
				return
			}
		}
	}
}

// Error might return errors risen in a Stage.
func (sh *StageHandler) Error() chan error {
	return sh.errChan
}

// Close this StageHandler and the current Stage.
func (sh *StageHandler) Close() error {
	close(sh.closeChan)
	return nil
}
