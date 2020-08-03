// Package bundle provides a library for interaction with Bundles as defined
// in the Bundle Protocol Version 7 (draft-ietf-dtn-bpbis-26.txt). This includes
// Bundle creation, modification, serialization and deserialization.
//
// The easiest way to create new Bundles is to use the BundleBuilder.
//
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
// Both serializing and deserializing bundles into the CBOR is supported.
//
//   // An existing Bundle bndl1 is serialized. The new bundle bndl2 is created
//   // from this. A common bytes.Buffer will be used.
//   buff := new(bytes.Buffer)
//   err1 := bndl1.WriteBundle(buff)
//   bndl2, err2 := bundle.ParseBundle(buff)
//
package bundle
