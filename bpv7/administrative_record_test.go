// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"testing"
)

func TestAdministrativeRecordManager_Register(t *testing.T) {
	arm := NewAdministrativeRecordManager()

	tests := []struct {
		name    string
		ar      AdministrativeRecord
		wantErr bool
	}{
		{"1st status report", &StatusReport{}, false},
		{"2nd status report", &StatusReport{}, true},
	}
	for _, test := range tests {
		if err := arm.Register(test.ar); (err != nil) != test.wantErr {
			t.Fatalf("%s: Register() error = %v, wantErr %v", test.name, err, test.wantErr)
		}
	}
}
