package wstunnel

import (
	"context"
	"net"
	"net/http"

	"nhooyr.io/websocket"
)

func Dial(ctx context.Context, u string, opts *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
	o := *opts
	o.Subprotocols = []string{TunnelWebSocketProtocol}
	return websocket.Dial(ctx, u, &o)
}

// DialerForGRPC can be used as an adapter between "ws"/"wss" URL scheme that the websocket library wants and
// gRPC target naming scheme.
func DialerForGRPC(readLimit int64, dialOpts *websocket.DialOptions) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, address string) (net.Conn, error) {
		conn, _, err := Dial(ctx, address, &websocket.DialOptions{
			HTTPClient:           dialOpts.HTTPClient,
			HTTPHeader:           dialOpts.HTTPHeader,
			CompressionMode:      dialOpts.CompressionMode,
			CompressionThreshold: dialOpts.CompressionThreshold,
		})
		if err != nil {
			return nil, err
		}
		if readLimit != 0 {
			conn.SetReadLimit(readLimit)
		}
		return websocket.NetConn(context.Background(), conn, websocket.MessageBinary), nil
	}
}
