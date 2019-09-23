package tcpcl

// This file contains code for the Client's termination state.

// terminate sends a SESS_TERM message to its peer and closes the session afterwards.
func (client *TCPCLClient) terminate(code SessionTerminationCode) {
	var sessTerm = NewSessionTerminationMessage(0, code)
	client.msgsOut <- &sessTerm

	if err := client.conn.Close(); err != nil {
		client.log().WithError(err).Warn("Failed to close TCP connection")
	} else {
		client.log().Info("Terminated session")
	}
}
