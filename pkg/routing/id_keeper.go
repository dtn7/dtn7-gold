// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"sync"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// idTuple is a tuple struct for looking up a bundle's iD - based on it's source
// node and DTN time part of the creation timestamp.
type idTuple struct {
	source bpv7.EndpointID
	time   bpv7.DtnTime
}

// newIdTuple creates an idTuple based on the given bundle.
func newIdTuple(bndl *bpv7.Bundle) idTuple {
	return idTuple{
		source: bndl.PrimaryBlock.SourceNode,
		time:   bndl.PrimaryBlock.CreationTimestamp.DtnTime(),
	}
}

// IdKeeper keeps track of the creation timestamp's sequence number for
// outbounding bundles.
type IdKeeper struct {
	data      map[idTuple]uint64
	mutex     sync.Mutex
	autoClean bool
}

// NewIdKeeper creates a new, empty IdKeeper.
func NewIdKeeper() IdKeeper {
	return IdKeeper{
		data:      make(map[idTuple]uint64),
		autoClean: true,
	}
}

// update updates the IdKeeper's state regarding this bundle and sets this
// bundle's sequence number.
func (idk *IdKeeper) update(bndl *bpv7.Bundle) {
	var tpl = newIdTuple(bndl)

	idk.mutex.Lock()
	if state, ok := idk.data[tpl]; ok {
		idk.data[tpl] = state + 1
	} else {
		idk.data[tpl] = 0
	}

	bndl.PrimaryBlock.CreationTimestamp[1] = idk.data[tpl]
	idk.mutex.Unlock()

	if idk.autoClean {
		idk.clean()
	}
}

// clean removes states which are older an hour and aren't the epoch time.
func (idk *IdKeeper) clean() {
	idk.mutex.Lock()

	var threshold = bpv7.DtnTimeNow() - 60*60*24

	for tpl := range idk.data {
		if tpl.time < threshold && tpl.time != bpv7.DtnTimeEpoch {
			delete(idk.data, tpl)
		}
	}
	idk.mutex.Unlock()
}
