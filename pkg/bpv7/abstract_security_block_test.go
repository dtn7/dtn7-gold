// SPDX-FileCopyrightText: 2021 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestAbstractSecurityBlock_CheckValid(t *testing.T) {
	ep, _ := NewEndpointID("dtn://test/")

	type fields struct {
		securityTargets           []uint64
		securityContextID         uint64
		securityContextFlags      uint64
		securitySource            EndpointID
		SecurityContextParameters []IDValueTuple
		securityResults           []TargetSecurityResults
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"a valid minimal ASB, should not error", fields{
			securityTargets:      []uint64{0},
			securityContextID:    SecConIdentBIBIOPHMACSHA,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleUInt64{
				id:    SecParIdBIBIOPHMACSHA2ShaVariant,
				value: HMAC256SHA256,
			}},
			securityResults: []TargetSecurityResults{{
				securityTarget: 0,
				results: []IDValueTuple{&IDValueTupleByteString{
					id:    SecConResultIDBIBIOPHMACSHA2ExpectedHMAC,
					value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
				}},
			}},
		}, false},
		{"a valid ASB, should not error", fields{
			securityTargets:      []uint64{0, 1, 2},
			securityContextID:    SecConIdentBIBIOPHMACSHA,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{
				&IDValueTupleByteString{
					id:    0,
					value: []byte{0, 0, 0, 0, 0},
				},
				&IDValueTupleByteString{
					id:    1,
					value: []byte{0, 0, 0, 0, 0},
				},
				&IDValueTupleByteString{
					id:    3,
					value: []byte{0, 0, 0, 0, 0},
				},
			},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{
						&IDValueTupleByteString{
							id:    0,
							value: []byte{0, 0, 0, 0, 0},
						},
						&IDValueTupleByteString{
							id:    1,
							value: []byte{0, 0, 0, 0, 0},
						},
					},
				},
				{
					securityTarget: 1,
					results: []IDValueTuple{
						&IDValueTupleByteString{
							id:    0,
							value: []byte{0, 0, 0, 0, 0},
						},
						&IDValueTupleByteString{
							id:    1,
							value: []byte{0, 0, 0, 0, 0},
						},
					},
				},
				{
					securityTarget: 2,
					results: []IDValueTuple{
						&IDValueTupleByteString{
							id:    0,
							value: []byte{0, 0, 0, 0, 0},
						},
						&IDValueTupleByteString{
							id:    1,
							value: []byte{0, 0, 0, 0, 0},
						},
					},
				},
			},
		}, false},
		{"not at least 1 entry in Security Targets, should error", fields{
			securityTargets:      []uint64{},
			securityContextID:    0,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{{
				securityTarget: 0,
				results: []IDValueTuple{&IDValueTupleByteString{
					id:    0,
					value: []byte{0, 0, 0, 0, 0},
				}},
			}},
		}, true},
		{"duplicate Security Target entries exist, should error", fields{
			securityTargets:      []uint64{0, 0},
			securityContextID:    0,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
			},
		}, true},
		{"number of entries in SecurityResults and SecurityTargets is not equal, should error", fields{
			securityTargets:      []uint64{0, 1, 2},
			securityContextID:    0,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 1,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
			},
		}, true},
		{"ordering of Security Targets and associated Security Results does not match, should error", fields{
			securityTargets:      []uint64{0, 1, 2},
			securityContextID:    0,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 2,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 1,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
			},
		}, true},
		{"Parameters Present Context Flag set, but no Security Parameter Context Field is present, should error", fields{
			securityTargets:      []uint64{0, 1},
			securityContextID:    0,
			securityContextFlags: 0x0,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 1,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
			},
		}, true},
		{"Parameters Present Context Flag set, but no Security Parameter Context Field is present, should error", fields{
			securityTargets:           []uint64{0, 1},
			securityContextID:         0,
			securityContextFlags:      0x1,
			securitySource:            ep,
			SecurityContextParameters: []IDValueTuple{},
			securityResults: []TargetSecurityResults{
				{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
				{
					securityTarget: 1,
					results: []IDValueTuple{&IDValueTupleByteString{
						id:    0,
						value: []byte{0, 0, 0, 0, 0},
					},
					},
				},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asb := &AbstractSecurityBlock{
				SecurityTargets:                      tt.fields.securityTargets,
				SecurityContextID:                    tt.fields.securityContextID,
				SecurityContextParametersPresentFlag: tt.fields.securityContextFlags,
				SecuritySource:                       tt.fields.securitySource,
				SecurityContextParameters:            tt.fields.SecurityContextParameters,
				SecurityResults:                      tt.fields.securityResults,
			}
			if err := asb.CheckValid(); (err != nil) != tt.wantErr {
				t.Errorf("CheckValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAbstractSecurityBlock_HasSecurityContextParametersPresentContextFlag(t *testing.T) {
	ep, _ := NewEndpointID("dtn://test/")
	type fields struct {
		securityTargets           []uint64
		securityContextID         uint64
		securityContextFlags      uint64
		securitySource            EndpointID
		SecurityContextParameters []IDValueTuple
		securityResults           []TargetSecurityResults
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"flag present", fields{
			securityTargets:      []uint64{0},
			securityContextID:    0,
			securityContextFlags: 0x1,
			securitySource:       ep,
			SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{
				id:    0,
				value: []byte{0, 0, 0, 0, 0},
			}},
			securityResults: []TargetSecurityResults{{
				securityTarget: 0,
				results: []IDValueTuple{&IDValueTupleByteString{
					id:    0,
					value: []byte{0, 0, 0, 0, 0},
				}},
			}},
		}, true},
		{"flag NOT present", fields{
			securityTargets:           []uint64{0},
			securityContextID:         0,
			securityContextFlags:      0x0,
			securitySource:            ep,
			SecurityContextParameters: []IDValueTuple{},
			securityResults: []TargetSecurityResults{{
				securityTarget: 0,
				results: []IDValueTuple{&IDValueTupleByteString{
					id:    0,
					value: []byte{0, 0, 0, 0, 0},
				}},
			}},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asb := &AbstractSecurityBlock{
				SecurityTargets:                      tt.fields.securityTargets,
				SecurityContextID:                    tt.fields.securityContextID,
				SecurityContextParametersPresentFlag: tt.fields.securityContextFlags,
				SecuritySource:                       tt.fields.securitySource,
				SecurityContextParameters:            tt.fields.SecurityContextParameters,
				SecurityResults:                      tt.fields.securityResults,
			}
			if got := asb.HasSecurityContextParametersPresentContextFlag(); got != tt.want {
				t.Errorf("HasSecurityContextParametersPresentContextFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetSecurityResultsCbor(t *testing.T) {
	tests := []struct {
		tsr1 TargetSecurityResults
	}{
		{TargetSecurityResults{
			securityTarget: 1,
			results: []IDValueTuple{
				&IDValueTupleByteString{
					id:    0,
					value: []byte{37, 35, 92, 90, 54},
				},
				&IDValueTupleByteString{
					id:    1,
					value: []byte{0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00},
				},
			},
		}},
		{TargetSecurityResults{
			securityTarget: 3,
			results: []IDValueTuple{
				&IDValueTupleByteString{

					id:    01,
					value: []byte{37, 35, 92, 90, 54},
				},
				&IDValueTupleByteString{

					id:    1,
					value: []byte{0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00},
				},
				&IDValueTupleByteString{

					id:    2,
					value: []byte{0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00},
				},
			},
		}},
	}

	for _, test := range tests {
		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.tsr1, buff); err != nil {
			t.Fatal(err)
		}

		tsr2 := TargetSecurityResults{}

		if err := cboring.Unmarshal(&tsr2, buff); err != nil {
			t.Fatalf("CBOR decoding failed: %v", err)
		}

		if !reflect.DeepEqual(test.tsr1, tsr2) {
			t.Fatalf("Target Security Resluts differ:\n%v\n%v", test.tsr1, tsr2)
		}
	}
}

func TestAbstractSecurityBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn://test/")
	tests := []struct {
		abs1 AbstractSecurityBlock
	}{
		{
			AbstractSecurityBlock{
				SecurityTargets:                      []uint64{0},
				SecurityContextID:                    SecConIdentBIBIOPHMACSHA,
				SecurityContextParametersPresentFlag: 0x1,
				SecuritySource:                       ep,
				SecurityContextParameters: []IDValueTuple{&IDValueTupleByteString{

					id:    SecParIdBIBIOPHMACSHA2WrappedKey,
					value: []byte{37, 35, 92, 90, 54},
				}},
				SecurityResults: []TargetSecurityResults{{
					securityTarget: 0,
					results: []IDValueTuple{&IDValueTupleByteString{

						id:    0,
						value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
					}},
				}},
			},
		},
		{
			AbstractSecurityBlock{
				SecurityTargets:                      []uint64{0, 1, 2},
				SecurityContextID:                    0,
				SecurityContextParametersPresentFlag: 0x0,
				SecuritySource:                       ep,
				SecurityContextParameters:            nil,
				SecurityResults: []TargetSecurityResults{
					{
						securityTarget: 0,
						results: []IDValueTuple{
							&IDValueTupleByteString{

								id:    0,
								value: []byte{37, 35, 92, 90, 54},
							},
							&IDValueTupleByteString{

								id:    1,
								value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
							},
						},
					},
					{
						securityTarget: 1,
						results: []IDValueTuple{
							&IDValueTupleByteString{

								id:    0,
								value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
							},
							&IDValueTupleByteString{

								id:    1,
								value: []byte{0, 0, 0, 0, 0, 37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
							},
						},
					},
					{
						securityTarget: 2,
						results: []IDValueTuple{
							&IDValueTupleByteString{

								id:    0,
								value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54, 0, 0, 0, 0, 0},
							},
							&IDValueTupleByteString{

								id:    1,
								value: []byte{0, 0, 37, 35, 92, 90, 54, 37, 35, 92, 90, 54, 0, 0, 0},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.abs1, buff); err != nil {
			t.Fatal(err)
		}

		abs2 := AbstractSecurityBlock{}
		if err := cboring.Unmarshal(&abs2, buff); err != nil {
			t.Fatalf("CBOR decoding failed: %v", err)
		}

		if !reflect.DeepEqual(test.abs1, abs2) {
			t.Fatalf("Abstract Security Blocs differ:\n%v\n%v", test.abs1, abs2)
		}
	}
}
