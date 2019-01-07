// Package bundle provides a library for interaction with Bundles as defined
// in the Bundle Protocol Version 7 (draft-ietf-dtn-bpbis-12.txt). This includes
// Bundle creation, modification, serialization and deserialization.
//
// New Bundles can be created by a combination of the NewBundle function with
// the NewPrimaryBlock and different New* functions for canonical blocks.
//
//   var bndl, err = NewBundle(
//     NewPrimaryBlock(
//       MustNotFragmented,
//       MustNewEndpointID("dtn", "dest"), MustNewEndpointID("dtn", "src"),
//       NewCreationTimestamp(DtnTimeEpoch, 0), 24 * 60 * 60),
//     []CanonicalBlock{
//       NewBundleAgeBlock(1, DeleteBundle, 0),
//       NewPayloadBlock(0, []byte("hello world!")),
//   })
//
// It's also possible to parse a serialized CBOR byte string into a new Bundle.
//
//   var bndl, err = bundle.NewBundleFromCbor(byteString)
//
// This package is still new and may be changed any time.
package bundle
