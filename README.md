# dtn7-go [![Build Status](https://travis-ci.org/dtn7/dtn7-go.svg?branch=master)](https://travis-ci.org/dtn7/dtn7-go) [![GoDoc](https://godoc.org/github.com/dtn7/dtn7-go?status.svg)](https://godoc.org/github.com/dtn7/dtn7-go)

Delay-Tolerant Networking software suite and library based on the Bundle
Protocol Version 7.


## Protocols
This software implements the current draft of the Bundle Protocol Version 7.

- Bundle Protocol Version 7 ([draft-ietf-dtn-bpbis-19.txt][dtn-bpbis-19])

### Convergence Layer
Bundles might be exchanged between nodes by the following protocols.

- TCP Convergence Layer Protocol Version 4
  ([draft-ietf-dtn-tcpclv4-14][dtn-tcpcl-14])
- Minimal TCP Convergence-Layer Protocol
  ([draft-ietf-dtn-mtcpcl-01.txt][dtn-mtcpcl-01])
- Bundle Broadcasting Connector, a generic Broadcasting Interface
    - [rf95modem] based CLA for LoRa PHY

### Routing
One of the following routing protocols might be used.

- Delay-Tolerant Link State Routing (DTLSR)
- Epidemic Routing
- Probabilistic Routing Protocol using History of Encounters and Transitivity
  (PRoPHET)
- Spray and Wait, vanilla and binary


## Software
### Installation

#### Package Manager

- Arch Linux: [*dtn7* (AUR)][aur-dtn7]
- Nix / NixOS: [*dtn7/nur-packages*][nur-dtn7] as a [NUR][nur]
- macOS: [jonashoechst/hoechst/dtn7][brew-dtn7] via [brew][brew]


#### From Source

Install the [Go programming language][golang], version 1.11 or later.

```bash
git clone https://github.com/dtn7/dtn7-go.git
cd dtn7-go

go build ./cmd/dtncat
go build ./cmd/dtnd
```


### dtnd
dtnd is a delay-tolerant networking daemon. It represents a node inside the
network and is able to transmit, receive and forward bundles to other nodes. A
node's neighbors may be specified in the configuration or detected within the
local network through a peer discovery. Bundles might be sent and received
through a REST-like web interface. The features and their configuration is
described inside the provided example
[`configuration.toml`][dtnd-configuration].

#### REST-API usage
The API might be used with the `dtncat` program or by plain HTTP requests.

```bash
# Create a outbounding bundle to dtn:host, containing "hello world"
# Payload must be base64 encoded
curl -d "{\"Destination\":\"dtn:host\", \"Payload\":\"`base64 <<< "hello world"`\"}" http://localhost:8080/send/

# Fetch received bundles. Payload is base64 encoded.
curl http://localhost:8080/fetch/
```

### dtncat
dtncat is a companion tool for dtnd and allows both sending and receiving
bundles through dtnd's REST-API.

```bash
$ ./dtncat help
dtncat [send|fetch|help] ...

dtncat send REST-API ENDPOINT-ID
  sends data from stdin through the given REST-API to the endpoint

dtncat fetch REST-API
  fetches all bundles from the given REST-API

Examples:
  dtncat send  "http://127.0.0.1:8080/" "dtn:alpha" <<< "hello world"
  dtncat fetch "http://127.0.0.1:8080/"
```


## Go Library
Multiple parts of this software are usable as a Go library. The `bundle`
package contains code for bundle modification, serialization and
deserialization and would most likely the most interesting part. If you are
interested in working with this code, check out the
[documentation][godoc].


[aur-dtn7]: https://aur.archlinux.org/packages/dtn7/
[dtn-bpbis-19]: https://tools.ietf.org/html/draft-ietf-dtn-bpbis-19
[dtn-mtcpcl-01]: https://tools.ietf.org/html/draft-ietf-dtn-mtcpcl-01
[dtn-tcpcl-14]: https://tools.ietf.org/html/draft-ietf-dtn-tcpclv4-14
[dtnd-configuration]: https://github.com/dtn7/dtn7-go/blob/master/cmd/dtnd/configuration.toml
[godoc]: https://godoc.org/github.com/dtn7/dtn7-go
[golang]: https://golang.org/
[nur-dtn7]: https://github.com/dtn7/nur-packages
[nur]: https://github.com/nix-community/NUR
[rf95modem]: https://github.com/gh0st42/rf95modem
[brew-dtn7]: https://github.com/jonashoechst/homebrew-hoechst/blob/master/dtn7.rb
[brew]: https://brew.sh
