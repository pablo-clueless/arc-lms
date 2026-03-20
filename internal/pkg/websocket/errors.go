package websocket

import "errors"

var (
	// ErrSendBufferFull is returned when the client's send buffer is full
	ErrSendBufferFull = errors.New("send buffer full")

	// ErrClientNotConnected is returned when trying to send to a disconnected client
	ErrClientNotConnected = errors.New("client not connected")
)
