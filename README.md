# dtn7-go [![Build Status](https://github.com/dtn7/dtn7-go/workflows/Build%20dtn7-go/badge.svg)](https://github.com/dtn7/dtn7-go/actions) [![GoDoc](https://godoc.org/github.com/dtn7/dtn7-go?status.svg)](https://godoc.org/github.com/dtn7/dtn7-go)

Delay-Tolerant Networking software suite and library based on the Bundle
Protocol Version 7.


## Protocols
This software implements the current draft of the Bundle Protocol Version 7.

- Bundle Protocol Version 7 ([draft-ietf-dtn-bpbis-17.txt][dtn-bpbis-17])

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


[dtn-bpbis-17]: https://tools.ietf.org/html/draft-ietf-dtn-bpbis-17
[dtn-mtcpcl-01]: https://tools.ietf.org/html/draft-ietf-dtn-mtcpcl-01
[dtn-tcpcl-14]: https://tools.ietf.org/html/draft-ietf-dtn-tcpclv4-14
[dtnd-configuration]: https://github.com/dtn7/dtn7-go/blob/master/cmd/dtnd/configuration.toml
[godoc]: https://godoc.org/github.com/dtn7/dtn7-go
[golang]: https://golang.org/
[rf95modem]: https://github.com/gh0st42/rf95modem
