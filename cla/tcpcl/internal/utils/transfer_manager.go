// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// TransferManager transfers Bundles bidirectionally.
//
// Therefore IncomingTransfer and OutgoingTransfer are generated automatically which will create msgs.Message.
type TransferManager struct {
	msgIn  <-chan msgs.Message
	msgOut chan<- msgs.Message

	chanBundles chan bundle.Bundle
	chanErrors  chan error

	segmentMtu uint64

	inTransfers sync.Map // map[uint64]*IncomingTransfer

	outNextId   uint64
	outFeedback sync.Map // map[uint64]chan msgs.Message

	stopChan chan struct{}
	stopped  uint64
}

// NewTransferManager for incoming and outgoing msgs.Message channels and a configured segment MTU.
func NewTransferManager(msgIn <-chan msgs.Message, msgOut chan<- msgs.Message, segmentMtu uint64) (tm *TransferManager) {
	tm = &TransferManager{
		msgIn:  msgIn,
		msgOut: msgOut,

		chanBundles: make(chan bundle.Bundle),
		chanErrors:  make(chan error),

		segmentMtu: segmentMtu,

		stopChan: make(chan struct{}),
	}

	go tm.handle()

	return
}

// Exchange channels for incoming Bundles or errors.
func (tm *TransferManager) Exchange() (bundles <-chan bundle.Bundle, errChan <-chan error) {
	bundles = tm.chanBundles
	errChan = tm.chanErrors
	return
}

// Close down this TransferManager.
func (tm *TransferManager) Close() (err error) {
	if atomic.CompareAndSwapUint64(&tm.stopped, 0, 1) {
		close(tm.stopChan)
	} else {
		err = fmt.Errorf("TransferManager was already closed")
	}

	return
}

func (tm *TransferManager) handle() {
	for {
		select {
		case <-tm.stopChan:
			return

		case msg := <-tm.msgIn:
			switch msg := msg.(type) {
			// Related to outgoing messages
			case *msgs.DataAcknowledgementMessage:
				if ackChan, ok := tm.outFeedback.Load(msg.TransferId); !ok {
					tm.chanErrors <- fmt.Errorf("received acknowledgement for unknown message %d", msg.TransferId)
					return
				} else {
					ackChan.(chan msgs.Message) <- msg
				}

			case *msgs.TransferRefusalMessage:
				if ackChan, ok := tm.outFeedback.Load(msg.TransferId); !ok {
					tm.chanErrors <- fmt.Errorf("received refusal for unknown message %d", msg.TransferId)
					return
				} else {
					ackChan.(chan msgs.Message) <- msg
				}

			// Related to incoming messages
			case *msgs.DataTransmissionMessage:
				transferI, _ := tm.inTransfers.LoadOrStore(msg.TransferId, NewIncomingTransfer(msg.TransferId))
				transfer := transferI.(*IncomingTransfer)

				if dam, err := transfer.NextSegment(msg); err != nil {
					tm.chanErrors <- err
					return
				} else {
					tm.msgOut <- dam
				}

				if transfer.IsFinished() {
					if b, err := transfer.ToBundle(); err != nil {
						tm.chanErrors <- err
						return
					} else {
						tm.chanBundles <- b
					}
					tm.inTransfers.Delete(msg.TransferId)
				}

			// Everything else
			default:
				tm.chanErrors <- fmt.Errorf("unexpected message %T", msg)
				return
			}
		}
	}
}

// Send an outgoing Bundle. This method blocks until the Bundle was sent successfully or an error arises.
func (tm *TransferManager) Send(b bundle.Bundle) error {
	transfer := NewBundleOutgoingTransfer(atomic.AddUint64(&tm.outNextId, 1)-1, b)

	ackChan := make(chan msgs.Message, 128)
	tm.outFeedback.Store(transfer.Id, ackChan)
	defer tm.outFeedback.Delete(transfer.Id)

	for {
		if atomic.LoadUint64(&tm.stopped) != 0 {
			return fmt.Errorf("TransferManager was stopped")
		}

		dtm, err := transfer.NextSegment(tm.segmentMtu)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return err
			}
		}

		tm.msgOut <- dtm

		select {
		case response := <-ackChan:
			switch response := response.(type) {
			case *msgs.DataAcknowledgementMessage:

			case *msgs.TransferRefusalMessage:
				return fmt.Errorf("received refusal message: %v", response)

			default:
				return fmt.Errorf("received unexpected message: %v", response)
			}

		case <-time.After(10 * time.Second):
			return fmt.Errorf("timeout: waiting for segment acknowledgement; id = %d, stopped = %t",
				transfer.Id, atomic.LoadUint64(&tm.stopped) != 0)
		}
	}

	return nil
}
