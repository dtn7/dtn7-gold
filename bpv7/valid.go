// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

// Valid is an interface with the CheckValid function. This function should
// return an errors for incorrect data. It should be implemented for the
// different types and sub-types of a Bundle. Each type is able to check its
// sub-types and by tree-like calls all errors of a whole Bundle can be
// detected.
// For non-trivial code, the multierror package might be used.
type Valid interface {
	// CheckValid returns an array of errors for incorrect data.
	CheckValid() error
}
