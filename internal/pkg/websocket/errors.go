package websocket

import "errors"

var (
	ErrSendBufferFull     = errors.New("send buffer full")
	ErrClientNotConnected = errors.New("client not connected")
)
