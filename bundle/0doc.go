// Package bundle provides a library for interaction with Bundles as defined
// in the Bundle Protocol Version 7 (draft-ietf-dtn-bpbis-13.txt). This includes
// Bundle creation, modification, serialization and deserialization.
//
// The easiest way to create new Bundles is to use the BundleBuilder.

//   bndl, err := bundle.Builder().
//     CRC(bundle.CRC32).
//     Source("dtn://src/").
//     Destination("dtn://dest/").
//     CreationTimestampNow().
//     Lifetime("30m").
//     HopCountBlock(64).
//     PayloadBlock([]byte("hello world!")).
//     Build()
//
// It's also possible to parse a serialized CBOR byte string into a new Bundle.
//
//   // TODO
//   var bndl, err = bundle.NewBundleFromCbor(byteString)
//
package bundle
