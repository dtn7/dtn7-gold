// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"sync"

	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// StageSetup wraps a Stage with two possible hooks (pre and post) to be used within the StageHandler.
type StageSetup struct {
	// Stage to be executed.
	Stage Stage

	// PreHook will be executed before starting the Stage, if not nil.
	PreHook func(*StageHandler, *State) error
	// PostHook will be executed after a finished Stage, if not nil.
	PostHook func(*StageHandler, *State) error
}

// StageHandler executes a sequence of Stages and passes the State from one Stage to another. Errors might be propagated
// back through the Error method.
type StageHandler struct {
	stages []StageSetup
	state  *State

	currentStage      StageSetup
	currentStageMutex sync.RWMutex

	errChan   chan error
	closeChan chan struct{}
}

// NewStageHandler for a slice of Stages, Message channels and a Configuration.
func NewStageHandler(stages []StageSetup, msgIn <-chan msgs.Message, msgOut chan<- msgs.Message, config Configuration) (sh *StageHandler) {
	sh = &StageHandler{
		stages: stages,
		state: &State{
			Configuration:  config,
			MsgIn:          msgIn,
			MsgOut:         msgOut,
			ExchangeMsgIn:  make(chan msgs.Message, 32),
			ExchangeMsgOut: make(chan msgs.Message, 32),
			StageError:     nil,
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
		sh.currentStage = StageSetup{}
		sh.currentStage.Stage = nil
		sh.currentStageMutex.Unlock()
	}()

	for i := 0; i < len(sh.stages); i++ {
		sh.currentStageMutex.Lock()
		sh.currentStage = sh.stages[i]
		sh.currentStageMutex.Unlock()

		if sh.currentStage.PreHook != nil {
			if err := sh.currentStage.PreHook(sh, sh.state); err != nil {
				sh.errChan <- err
				return
			}
		}

		sh.currentStage.Stage.Handle(sh.state, sh.closeChan)
		if err := sh.state.StageError; err != nil {
			sh.errChan <- err
			return
		}

		if sh.stages[i].PostHook != nil {
			if err := sh.stages[i].PostHook(sh, sh.state); err != nil {
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

// Exchanges returns two channels for Message exchange with the peer.
func (sh *StageHandler) Exchanges() (incoming <-chan msgs.Message, outgoing chan<- msgs.Message) {
	return sh.state.ExchangeMsgIn, sh.state.ExchangeMsgOut
}

// Close this StageHandler and the current Stage.
func (sh *StageHandler) Close() error {
	close(sh.closeChan)
	return nil
}
