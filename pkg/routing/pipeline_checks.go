// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"errors"
)

// CheckFunc is used within the Pipeline for bpv7.Bundle / BundleDescriptor inspection.
type CheckFunc func(*Pipeline, BundleDescriptor) error

// CheckRouting wraps the Algorithm.DispatchingAllowed method.
func CheckRouting(pipeline *Pipeline, descriptor BundleDescriptor) (err error) {
	if pipeline.Algo.DispatchingAllowed(descriptor) {
		err = errors.New("routing algorithm prohibited dispatching")
	}
	return
}
