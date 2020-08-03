// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

// coreError is a simple error-struct.
type coreError struct {
	msg string
}

// newCoreError creates a new coreError with the given message.
func newCoreError(msg string) *coreError {
	return &coreError{msg}
}

func (e coreError) Error() string {
	return e.msg
}
