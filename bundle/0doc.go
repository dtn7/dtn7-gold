// Package bundle provides a library for interaction with Bundles as defined
// in the Bundle Protocol Version 7 (draft-ietf-dtn-bpbis-12.txt). This includes
// Bundle creation, modification, serialization and deserialization.
//
// New Bundles can be created by a combination of the NewBundle function with
// the NewPrimaryBlock and different New* functions for canonical blocks.
//
// var bndl, err = bundle.NewBundle(
//   bundle.NewPrimaryBlock(
//     bundle.MustNotFragmented,
//     bundle.MustNewEndpointID("dtn", "dest"),
//     bundle.MustNewEndpointID("dtn", "src"),
//     bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 24*60*60),
//   []bundle.CanonicalBlock{
//       bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
//       bundle.NewPayloadBlock(0, []byte("hello world!")),
//   })
//
// It's also possible to parse a serialized CBOR byte string into a new Bundle.
//
//   var bndl, err = bundle.NewBundleFromCbor(byteString)
//
// This package is still new and may be changed any time.
package bundle
