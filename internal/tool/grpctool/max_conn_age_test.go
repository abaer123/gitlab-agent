package grpctool

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool/test"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/wstunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"nhooyr.io/websocket"
)

// These tests verify our understanding of how MaxConnectionAge and MaxConnectionAgeGrace work in gRPC
// and that our WebSocket tunneling works fine with it.

func TestMaxConnectionAge_gRPC(t *testing.T) {
	testKeepalive(t, false, testClient)
}

func TestMaxConnectionAge_WebSocket(t *testing.T) {
	testKeepalive(t, true, testClient)
}

func testClient(t *testing.T, client test.TestingClient) {
	start := time.Now()
	resp, err := client.StreamingRequestResponse(context.Background())
	require.NoError(t, err)
	_, err = resp.Recv()
	require.Equal(t, io.EOF, err, "%s. Elapsed: %s", err, time.Since(start))
}

func testKeepalive(t *testing.T, websocket bool, f func(*testing.T, test.TestingClient)) {
	t.Parallel()
	maxAge := 3 * time.Second
	l, dial := listenerAndDialer(websocket)
	defer func() {
		assert.NoError(t, l.Close())
	}()
	s := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      maxAge,
			MaxConnectionAgeGrace: maxAge,
		}),
	)
	defer s.Stop()
	test.RegisterTestingServer(s, &test.GrpcTestingServer{
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			//start := time.Now()
			//ctx := server.Context()
			//<-ctx.Done()
			//t.Logf("ctx.Err() = %v after %s", ctx.Err(), time.Since(start))
			//return ctx.Err()
			time.Sleep(maxAge + maxAge*2/10) // +20%
			return nil
		},
	})
	go func() {
		assert.NoError(t, s.Serve(l))
	}()
	conn, err := grpc.DialContext(
		context.Background(),
		"ws://pipe",
		grpc.WithContextDialer(dial),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, conn.Close())
	}()
	f(t, test.NewTestingClient(conn))
}

func listenerAndDialer(webSocket bool) (net.Listener, func(context.Context, string) (net.Conn, error)) {
	l := NewDialListener()
	if !webSocket {
		return l, l.DialContext
	}
	lisWrapper := wstunnel.ListenerWrapper{}
	lWrapped := lisWrapper.Wrap(l)
	return lWrapped, wstunnel.DialerForGRPC(0, &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return l.DialContext(ctx, addr)
				},
			},
		},
	})
}
