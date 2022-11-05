// SPDX-FileCopyrightText: 2019, 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package bpv7 provides a library for interaction with Bundles as defined
// in the Bundle Protocol Version 7 (draft-ietf-dtn-bpbis-31.txt). This includes
// Bundle creation, modification, serialization and deserialization.
//
// The easiest way to create new Bundles is to use the BundleBuilder.
//
//	bundle, err := bpv7.Builder().
//	  CRC(bpv7.CRC32).
//	  Source("dtn://src/").
//	  Destination("dtn://dest/").
//	  CreationTimestampNow().
//	  Lifetime(time.Hour).
//	  HopCountBlock(64).
//	  PayloadBlock([]byte("hello world!")).
//	  Build()
//
// Both serializing and deserializing bundles into the CBOR is supported.
//
//	// An existing Bundle b1 is serialized. The new bundle b2 is created
//	// from this. A common bytes.Buffer will be used.
//	buff := new(bytes.Buffer)
//	err1 := b1.WriteBundle(buff)
//	b2, err2 := bpv7.ParseBundle(buff)
package bpv7
