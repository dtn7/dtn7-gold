<!--
SPDX-FileCopyrightText: 2021 Markus Sommer

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Daemon Configuration

Here, we'll go over the [sample config-file](https://github.com/dtn7/dtn7-go/blob/master/cmd/dtnd/configuration.toml) and explain the available options in-depth.

## [core]

### store

Path to the on-disk store used to persis bundles, may be either a relative or absolute path.

The store consist of a [badgerhold](https://github.com/timshannon/badgerhold) database for metadata, the bundle contents are written directly to disk.

### inspect-all-bundles

Hell if I know what that does....

### node-id

Public name of the node. Will be set as the default Endoint id for all CLAs, unless a different one has been speified.

According to the [standard](https://tools.ietf.org/html/draft-ietf-dtn-bpbis-31#section-4.2.5.1.1), your ID should be:

- `dtn:none`, if you are actually nothing
- `dtn://` + the actual name + `/` <- for a unicast enpoint
- `dtn://~` + the name of your multicast group + `/`  <- for something that is not a unicast enpoint (it does not neccessarily have to be a multicast group, but that's the only use case I can think off the top of my head)

You might also use the [other](https://tools.ietf.org/html/draft-ietf-dtn-bpbis-31#section-4.2.5.1.2) adressing scheme, but we can't guarantee that such behaviour won't lead to death and/or dismemberment.

## [logging]

## [discovery]

## [agents]

## [[listen]]

## [[peer]]

## [routing]

Select the routing algorithm, you can choose on from the list `["epidemic", "spray", "binary_spray", "dtlsr", "prophet", "sensor-mule"]`

### epidemic

*Epidemic Routing* is the simplest delay-tolerant routing algorithm.
Alls bundles are always sent to all peers, which gives the best delivery probabilty, but also the highest overhead.
Note that `dtnd` keeps track of peers who have already been sent a specific bundle, so each bundle should nly be forwarded to each peer once.

### spray

The *Spray & Wait* routing algorithm.

### binary_spray

The *Binary Spray & Wait* routing algorithm.

### dtlsr

The *Delay-Tolerant Link-State Routing* algorithm.

### prophet

The *PRoPHET* routing algorithm.

### sensor-mule

For dtn over equine carriers.