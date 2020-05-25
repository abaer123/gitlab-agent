package wstunnel

import "net/http"

const (
	// TunnelWebSocketProtocol is a subprotocol that allows client and server recognize each other.
	// See https://tools.ietf.org/html/rfc6455#section-11.3.4.
	TunnelWebSocketProtocol = "ws-tunnel"
	webSocketProtocolHeader = "Sec-WebSocket-Protocol"
)

func ProtocolHeader() http.Header {
	header := http.Header{}
	header.Set(webSocketProtocolHeader, TunnelWebSocketProtocol)
	return header
}
