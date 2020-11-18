// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"errors"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// pipeline_check contains check functions to be applied within the Pipeline's preprocessing.

// CheckFunc is used within the Pipeline for bpv7.Bundle / BundleDescriptor inspection.
type CheckFunc func(*Pipeline, BundleDescriptor) error

// CheckRouting wraps the Algorithm.DispatchingAllowed method.
func CheckRouting(pipeline *Pipeline, descriptor BundleDescriptor) (err error) {
	if pipeline.Algorithm.DispatchingAllowed(descriptor) {
		err = errors.New("routing algorithm prohibited dispatching")
	}
	return
}

// CheckLifetime of the bpv7.Bundle.
func CheckLifetime(_ *Pipeline, descriptor BundleDescriptor) (err error) {
	if descriptor.MustBundle().IsLifetimeExceeded() {
		err = errors.New("bundle lifetime is exceeded")
	}
	return
}

// CheckHopCount of an optionally exceeded HopCountBlock.
func CheckHopCount(_ *Pipeline, descriptor BundleDescriptor) (err error) {
	if block, blockErr := descriptor.MustBundle().ExtensionBlock(bpv7.ExtBlockTypeHopCountBlock); blockErr == nil {
		if block.Value.(*bpv7.HopCountBlock).IsExceeded() {
			err = errors.New("hop count block is exceeded")
		}
	}
	return
}
