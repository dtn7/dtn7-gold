<!--
SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
SPDX-FileCopyrightText: 2020 Jonas Höchst
SPDX-FileCopyrightText: 2020 Matthias Axel Kröll

SPDX-License-Identifier: GPL-3.0-or-later
-->

# dtn7-go
[![Release](https://img.shields.io/github/v/tag/dtn7/dtn7-go?label=version)](https://github.com/dtn7/dtn7-go/releases)
[![GoDoc](https://godoc.org/github.com/dtn7/dtn7-go?status.svg)](https://godoc.org/github.com/dtn7/dtn7-go)
[![CI](https://github.com/dtn7/dtn7-go/workflows/CI/badge.svg)](https://github.com/dtn7/dtn7-go/actions)
[![REUSE status](https://api.reuse.software/badge/github.com/dtn7/dtn7-go)](https://api.reuse.software/info/github.com/dtn7/dtn7-go)

Delay-Tolerant Networking software suite and library based on the Bundle Protocol Version 7.


## Protocols
This software implements the current draft of the Bundle Protocol Version 7.

- Bundle Protocol Version 7 ([draft-ietf-dtn-bpbis-26][dtn-bpbis-26])

### Convergence Layer
Bundles might be exchanged between nodes by the following protocols.

- TCP Convergence Layer Protocol Version 4 ([draft-ietf-dtn-tcpclv4-14][dtn-tcpcl-14])
- Minimal TCP Convergence-Layer Protocol ([draft-ietf-dtn-mtcpcl-01][dtn-mtcpcl-01])
- Bundle Broadcasting Connector, a generic Broadcasting Interface
    - [rf95modem] based CLA for LoRa PHY by [rf95modem-go]

### Routing
One of the following routing protocols might be used.

- Delay-Tolerant Link State Routing (DTLSR)
- Epidemic Routing
- Probabilistic Routing Protocol using History of Encounters and Transitivity (PRoPHET)
- Sensor Network specific routing algorithm for data mules, [documentation](sensor-network-mule-documentation)
- Spray and Wait, vanilla and binary


## Software
### Installation

#### Package Manager

- Arch Linux: [_dtn7_][aur-dtn7] ([AUR][arch-aur])
- macOS: [_jonashoechst/hoechst/dtn7_][brew-dtn7] ([brew][brew])
- Nix / NixOS: [_dtn7/nur-packages_][nur-dtn7] ([NUR][nixos-nur])


#### From Source

Install the [Go programming language][golang], version 1.13 or later.

```bash
git clone https://github.com/dtn7/dtn7-go.git
cd dtn7-go

go build ./cmd/dtn-tool
go build ./cmd/dtnd
```


### dtnd
`dtnd` is a delay-tolerant networking daemon.
It represents a node inside the network and is able to transmit, receive and forward bundles to other nodes.
A node's neighbors may be specified in the configuration or detected within the local network through a peer discovery.
Bundles might be sent and received through a REST-like web interface.
The features and their configuration is described inside the provided example [`configuration.toml`][dtnd-configuration].

#### REST API / WebSocket API
Different interfaces are provided to allow communication from external programs with `dtnd`.
More precisely: a REST API and a WebSocket API.

The simpler REST API allows a client to register itself with an address, receive bundles and create / dispatch new ones.
This is made by POSTing JSON objects to `dtnd`'s RESTful HTTP server.
The endpoints and structure of the JSON objects are described in the [documentation][godoc] for the `github.com/dtn7/dtn7-go/agent.RestAgent` type.

However, a bidirectional communication is possible via the WebSocket API.
This API sends CBOR-encoded messages.
The details can be found in the `ws_agent`-files of the `agent` package.
But one can also simply use it with the `github.com/dtn7/dtn7-go/agent.WebSocketAgentConnector`, which implements a client.

### dtn-tool
A ready-to-use program that utilizes the WebSocket API mentioned above is `dtn-tool`, a _swiss army knife_ for bundles.

It allows the simple creation of new bundles, written to a file or the stdout.
Furthermore, one can print out bundles as a human / script readable JSON object.
To exchange bundles, `dtn-tool` might _watch_ a directory and send all new bundle files to the corresponding `dtnd` instance.
In the same way, incoming bundles from `dtnd` are stored in this directory.

```
Usage of ./dtn-tool create|show|exchange:

./dtn-tool create sender receiver -|filename [-|filename]
  Creates a new Bundle, addressed from sender to receiver with the stdin (-)
  or the given file (filename) as payload. If no further specified, the
  Bundle is stored locally named after the hex representation of its ID.
  Otherwise, the Bundle can be written to the stdout (-) or saved
  according to a freely selectable filename.

./dtn-tool show -|filename
  Prints a JSON version of a Bundle, read from stdin (-) or filename.

./dtn-tool exchange websocket endpoint-id directory
  ./dtn-tool registeres itself as an agent on the given websocket and writes
  incoming Bundles in the directory. If the user dropps a new Bundle in the
  directory, it will be sent to the server.

```


## Go Library
Multiple parts of this software are usable as a Go library.
The `bundle` package contains code for bundle modification, serialization and deserialization and would most likely the most interesting part.
If you are interested in working with this code, check out the [documentation][godoc].


## Contributing
Contributions will receive warmhearted welcome.

[Gofmt][gofmt] must be used for formatting the source.
Further inspection of the code via [golangci-lint][golangci-lint] is highly recommended.

As a development environment you may, of course, use whatever you personally like best.
However, we have had good experience with [GoLand][goland], especially because of the size of the project.

Assuming you have a supported version of the [Go programming language][golang] installed, just clone the repository and install the dependencies as documented in the _Installation, From Source_ section above.

Please document your changes both in good commit messages and within the [CHANGELOG.md][CHANGELOG.md] file.

Also, an attempt is made to be [REUSE][reuse] compliant.
For automatic copyright header generation, the `contrib/reuse/reuse-headers.py` script exists.

### OS-specific
#### macOS
Installing the [Go programming language][golang] via [brew][brew], should solve permission errors while trying to fetch the dependencies.


## License

This project's code is licensed under the [GNU General Public License version 3 (_GPL-3.0-or-later_)][license-gpl3].
To simplify the copyright stuff, the [REUSE][reuse] tool is used.


[CHANGELOG.md]: CHANGELOG.md
[arch-aur]: https://wiki.archlinux.org/index.php/Arch_User_Repository
[aur-dtn7]: https://aur.archlinux.org/packages/dtn7/
[brew-dtn7]: https://github.com/jonashoechst/homebrew-hoechst/blob/master/dtn7.rb
[brew]: https://brew.sh
[dtn-bpbis-26]: https://tools.ietf.org/html/draft-ietf-dtn-bpbis-26
[dtn-mtcpcl-01]: https://tools.ietf.org/html/draft-ietf-dtn-mtcpcl-01
[dtn-tcpcl-14]: https://tools.ietf.org/html/draft-ietf-dtn-tcpclv4-14
[dtnd-configuration]: https://github.com/dtn7/dtn7-go/blob/master/cmd/dtnd/configuration.toml
[godoc]: https://godoc.org/github.com/dtn7/dtn7-go
[gofmt]: https://blog.golang.org/gofmt
[goland]: https://www.jetbrains.com/go/
[golang]: https://golang.org/
[golangci-lint]: https://github.com/golangci/golangci-lint
[license-gpl3]: LICENSES/GPL-3.0-or-later.txt
[nixos-nur]: https://github.com/nix-community/NUR
[nur-dtn7]: https://github.com/dtn7/nur-packages
[reuse]: https://reuse.software/
[rf95modem-go]: https://github.com/dtn7/rf95modem-go
[rf95modem]: https://github.com/gh0st42/rf95modem
[sensor-network-mule-documentation]: https://godoc.org/github.com/dtn7/dtn7-go/core#SensorNetworkMuleRouting


<!-- vim: set ts=2 ft=markdown spell: -->
