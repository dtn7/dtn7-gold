// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"sync"

	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// StageHandler executes a sequence of Stages and passes the State from one Stage to another. Errors might be propagated
// back through the Error method.
type StageHandler struct {
	stages []Stage
	state  *State

	currentStage      Stage
	currentStageMutex sync.RWMutex

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

	defer func() {
		sh.currentStageMutex.Lock()
		sh.currentStage = nil
		sh.currentStageMutex.Unlock()
	}()

	for i := 0; i < len(sh.stages); i++ {
		sh.currentStageMutex.Lock()
		sh.currentStage = sh.stages[i]
		sh.currentStage.Start(sh.state)
		sh.currentStageMutex.Unlock()

		select {
		case <-sh.closeChan:
			_ = sh.currentStage.Close()
			return

		case <-sh.currentStage.Finished():
			if err := sh.state.StageError; err != nil {
				sh.errChan <- err
				return
			}
		}
	}
}

// Error might return errors risen in a Stage.
func (sh *StageHandler) Error() <-chan error {
	return sh.errChan
}

// Exchanges returns two optional channels for Message exchange with the peer.
//
// The implementation for a StageHandler wraps the call for the current Stage. If there is currently no stage,
// exchangeOk is always false.
func (sh *StageHandler) Exchanges() (outgoing chan<- msgs.Message, incoming <-chan msgs.Message, exchangeOk bool) {
	sh.currentStageMutex.RLock()
	defer sh.currentStageMutex.RUnlock()

	if sh.currentStage == nil {
		exchangeOk = false
		return
	} else {
		return sh.currentStage.Exchanges()
	}
}

// Close this StageHandler and the current Stage.
func (sh *StageHandler) Close() error {
	close(sh.closeChan)
	return nil
}
