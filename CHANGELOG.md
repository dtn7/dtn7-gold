<!--
SPDX-FileCopyrightText: 2020 Alvar Penning

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keep-a-changelog], and this
project adheres to [Semantic Versioning][semantic-versioning].

<!--
Types of changes:

- Added       for new features.
- Changed     for changes in existing functionality.
- Deprecated  for soon-to-be removed features.
- Removed     for now removed features.
- Fixed       for any bug fixes.
- Security    in case of vulnerabilities
-->

## [Unreleased]
### Added
- REUSE compliance and a new copyright header generation script.
- Sort Bundle's CanonicalBlocks on creation and block modification.
- Custom SignatureBlock for cryptographic ed25519 Bundle signatures.
- dtnd supports attaching a SignatureBlock for outgoing Bundles.

### Changed
- Replaced TravisCI with GitHub Actions.
- List all Extension Block type codes in `bundle/extension_block.go`.
- Fragmentation tries to copy the original CRC type.
- BundleBuilder: Add replicate block flag for Bundle Age, Hop Count, and
  Previous Node Blocks to ease Bundle fragmentation.

### Removed
- Drop compatibility with Go versions below 1.13.

### Fix
- Ensure only Payload Blocks are allowed to get Block Number 1 when
  adding Extension Blocks to an empty Bundle.
- PrimaryBlock: always overwrite CRC, don't rely on cached values


## [0.7.1] - 2020-08-03
### Fixed
- Restoring compatibility with Go 1.11.

### Deprecated
- Compatibility with Go 1.11 will soon be dropped. Expect the 0.7.x
  releases the last ones to support this now almost two year old Go
  version. Sorry.


## [0.7.0] - 2020-08-03
### Added
- `AdministrativeRecordManager` to allow more dynamic Administrative
  Records.
- `EndpointID` gets a singleton property, _ietf-dtn-bpbis-26_.
- Make repository more friendly for new contributors by
    - GitHub Issue template,
    - `CHANGELOG.md` file, and
    - Contributing section in `README.md`.

### Changed
- Enforce strict `dtn` URI scheme based on the ABNF, like this
  `dtn://NODE-NAME/OPTIONAL-VARIOUS-CHARS`.
- Allow peer discovery to work with multiple Endpoint IDs.
- CLA management is performed by the CLA Manager.
- Time is normed to milliseconds, _ietf-dtn-bpbis-26_:
    - DTN Time: milliseconds instead of seconds
    - Primary Block's lifetime: milliseconds instead of microseconds
    - Bundle Age Block: milliseconds instead of microseconds
- Bump draft-ietf-dtn-bpbis version from 24 to 26.

### Fixed
- `BundleBuilder` sorts CanonicalBlocks based on their block number.


## [0.6.1] - 2020-04-16
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Normalize RestAgent's error response field

Previously, the _error_ reporting in the JSON objects were differerent
for different methods. This makes a programming library unnecessarily
complicated.

Furthermore, the bundle/arecord package was moved into bundle. This was
actually only supposed to happen in the next main release.


## [0.6.0] - 2020-04-16
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

New agent package for different clients in dtnd

Some changes have built up for this release. The biggest change is the
new agent package, which replaces the old REST interface.

Further changes in headwords:
- agent: MuxAgent to multiplex "child" agents
- agent: PingAgent to respond to pings


## [0.5.4] - 2020-01-04
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Different bug-fixes for bundle package and store

The memory footprint of the store has been reduced so it runs smoothly
on smaller platforms. Additionally the deserialization of bundles was
examined via fuzzing. As a result, two critical bugs were fixed.


## [0.5.3] - 2019-12-17
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Optimize BBC for rf95modem

- Recipients can report failed transmissions. This leads to cancellation
  with later retransmission.
- xz compress Bundles
- Fix other bugs..


## [0.5.2] - 2019-12-06
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Bundle Fragmentation

The bundle package now supports Bundle fragmentation and reassembly
regarding a given MTU.


## [0.5.1] - 2019-12-03
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

TCPCL bugfixes, wider ExtensionBlock serialization

This small release fixes several critical bugs in the TCP Convergence
Layer that caused crashes when reconnecting. More work on this CLA is
still neccessary.

Furthermore, the ExtensionBlock interface has been extended. Thus it is
now possible to serialize the block-type specific data of a
CanonicalBlock not only to CBOR, but to generic binary data. Instead of
the cboring.CborMarshaler it is now possible to implement the
encoding.Binary{Marshaler,Unmarshaler}. Based on the implemented
interface, a serialization is chosen.

## [0.5.0] - 2019-11-08
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

LoRa-based CLA and update BP to dtn-bpbis-17


## [0.4.0] - 2019-10-11
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

TCPCL and PRoPHET

This release implements the Delay-Tolerant Networking TCP Convergence
Layer Protocol Version 4 for a bidirectional Bundle exchange.
Furthermore, the PRoPHET routing protocol was added.


## [0.3.0] - 2019-09-06
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

DTLSR, MTCP Keep Alive

The Delay Tolerant Link State Routing protocol was implemented.
Furthermore, a TCP keep alive was added to MTCP against link failure.


## [0.2.1] - 2019-08-08
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Update Bundle Protocol Version 7 to draft 14

The most significant change is the establishment of a mandatory CRC
value for the Primary Block. Furthermore, the Manifest Block that was
previously marked as reserved is now removed.


## [0.2.0] - 2019-08-02
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

New release

- Cron Service
- Manage Bundle IDs
- New CLA/Convergence Management
- New routing algorithms: Spray and Wait, Binary Spray and Wait
- Redesigned Store for meta data and with lazy Bundle loading


## [0.1.1] - 2019-07-09
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

Refactored bundle package.

Mostly replacing codec library with new cboring library for CBOR
serialization, resulting in a major speedup.


## [0.1.0] - 2019-06-06
> _This release was created before adapting the
> [Keep a Changelog][keep-a-changelog] format._

> _The date of this release may be incorrect because the tag was added
> after switching to Semantic Versioning._

First, unstable release


[keep-a-changelog]: https://keepachangelog.com/en/1.0.0/
[semantic-versioning]: https://semver.org/spec/v2.0.0.html

[0.1.0]: https://github.com/dtn7/dtn7-go/releases/tag/v0.1.0
[0.1.1]: https://github.com/dtn7/dtn7-go/compare/v0.1.0...v0.1.1
[0.2.0]: https://github.com/dtn7/dtn7-go/compare/v0.1.1...v0.2.0
[0.2.1]: https://github.com/dtn7/dtn7-go/compare/v0.2.0...v0.2.1
[0.3.0]: https://github.com/dtn7/dtn7-go/compare/v0.2.1...v0.3.0
[0.4.0]: https://github.com/dtn7/dtn7-go/compare/v0.3.0...v0.4.0
[0.5.0]: https://github.com/dtn7/dtn7-go/compare/v0.4.0...v0.5.0
[0.5.1]: https://github.com/dtn7/dtn7-go/compare/v0.5.0...v0.5.1
[0.5.2]: https://github.com/dtn7/dtn7-go/compare/v0.5.1...v0.5.2
[0.5.3]: https://github.com/dtn7/dtn7-go/compare/v0.5.2...v0.5.3
[0.5.4]: https://github.com/dtn7/dtn7-go/compare/v0.5.3...v0.5.4
[0.6.0]: https://github.com/dtn7/dtn7-go/compare/v0.5.4...v0.6.0
[0.6.1]: https://github.com/dtn7/dtn7-go/compare/v0.6.0...v0.6.1
[0.7.0]: https://github.com/dtn7/dtn7-go/compare/v0.6.1...v0.7.0
[0.7.1]: https://github.com/dtn7/dtn7-go/compare/v0.7.0...v0.7.1
[Unreleased]: https://github.com/dtn7/dtn7-go/compare/v0.7.1...master


<!-- vim: set tw=72 colorcolumn=72 ts=2 ft=markdown spell: -->
