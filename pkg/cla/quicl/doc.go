// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

/*
Package quicl implements an experimental QUIC convergence layer.
Note that this convergence layer is not part of the Bundle Protocol or its associated specifications.


Why?
The bundle protocol's "native" CLAs come with several significant downsides.

MTCP is simple but very limited in its functionality.
Most significantly, even though it uses a bidirectional TCP connection, the CLA's communication is unidirectional.

TCPCL is more powerful but also very complicated.
It has a multi-step handshake and requires the implementer to do some heavy lifting.

QUICL is meant as a reasonable middle ground between these extremes.
While QUIC also has an extensive handshake and powerful features (e.g. data multiplexing),
this work has already been done if one uses an existing QUIC library.


Protocol
When it comes to the establishment of a connection, there are two distinct roles.
The listener waits for incoming connections and spawns a new "endpoint"-goroutine each time a dialer connects.

Once the connection has been established, the endpoints perform a simple handshake, exchanging EndpointIDs.
While, in most cases, the dialer's endpoint *should* know the listener's endpoint's ID,
we cannot rely on this always being the case.
We have decided against distinguishing these cases during the handshake for simplicities' sake,
and both endpoints will always exchange IDs.

The listener's endpoint waits for a stream to be opened on the QUIC connection and receives the dialer's EndpointID.
If the dialer does not initiate the handshake within a set time,
the listener closes the connection with error code 4 (PeerError).

If the listener cannot unmarshal the dialer's ID,
it closes the QUIC connection with application error code 4 (PeerError).
Otherwise, if the dialer's ID was received correctly, the listener sends their own ID.
Endpoint IDs are serialised using their own native CBOR marshallers.


Bundle transmission

QUIC allows for the simultaneous sending and receiving of multiple streams of data on the same connection,
with the QUIC library handling (de-)multiplexing of data.
This greatly simplifies bundle transmission since we don't need to track any state ourselves.
If a node wants to transmit a bundle, it simply opens a new stream and sends the serialised bundle data.
On the receiving side, when a node notices a new stream opening,
it launches a new handler goroutine which receives and deserialises the bundle and terminates.
A single stream will always carry exactly one bundle and be closed after the transmission is completed.
*/

package quicl
