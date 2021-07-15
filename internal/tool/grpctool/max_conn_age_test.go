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
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"nhooyr.io/websocket"
)

func TestMaxConnectionAge2MaxPollDuration(t *testing.T) {
	const maxConnAge = time.Minute
	x := MaxConnectionAge2MaxPollDuration(maxConnAge)
	assert.Less(t, x*2, maxConnAge)
}

// These tests verify our understanding of how MaxConnectionAge and MaxConnectionAgeGrace work in gRPC
// and that our WebSocket tunneling works fine with it.

func TestMaxConnectionAge(t *testing.T) {
	t.Parallel()
	const maxAge = 3 * time.Second
	srv := &test.GrpcTestingServer{
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			//start := time.Now()
			//ctx := server.Context()
			//<-ctx.Done()
			//t.Logf("ctx.Err() = %v after %s", ctx.Err(), time.Since(start))
			//return ctx.Err()
			time.Sleep(maxAge + maxAge*2/10) // +20%
			return nil
		},
	}
	testClient := func(t *testing.T, client test.TestingClient) {
		start := time.Now()
		resp, err := client.StreamingRequestResponse(context.Background())
		require.NoError(t, err)
		_, err = resp.Recv()
		require.Equal(t, io.EOF, err, "%s. Elapsed: %s", err, time.Since(start))
	}
	kp := keepalive.ServerParameters{
		MaxConnectionAge:      maxAge,
		MaxConnectionAgeGrace: maxAge,
	}
	t.Run("gRPC", func(t *testing.T) {
		testKeepalive(t, false, kp, srv, testClient)
	})
	t.Run("WebSocket", func(t *testing.T) {
		testKeepalive(t, true, kp, srv, testClient)
	})
}

func TestMaxConnectionAgeAndMaxPollDuration(t *testing.T) {
	t.Parallel()
	const maxAge = 3 * time.Second
	srv := &test.GrpcTestingServer{
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			time.Sleep(MaxConnectionAge2MaxPollDuration(maxAge))
			return nil
		},
	}
	testClient := func(t *testing.T, client test.TestingClient) {
		for i := 0; i < 3; i++ {
			start := time.Now()
			resp, err := client.StreamingRequestResponse(context.Background())
			require.NoError(t, err)
			_, err = resp.Recv()
			require.Equal(t, io.EOF, err, "%s. Elapsed: %s", err, time.Since(start))
		}
	}

	kp := maxConnectionAge2GrpcKeepalive(maxAge)
	t.Run("gRPC", func(t *testing.T) {
		testKeepalive(t, false, kp, srv, testClient)
	})
	t.Run("WebSocket", func(t *testing.T) {
		testKeepalive(t, true, kp, srv, testClient)
	})
}

func TestMaxConnectionAgeAndMaxPollDurationRandomizedSequential(t *testing.T) {
	t.Parallel()
	const maxAge = 3 * time.Second
	srv := &test.GrpcTestingServer{
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			time.Sleep(time.Duration(rand.Int63nRange(0, int64(MaxConnectionAge2MaxPollDuration(maxAge)))))
			return nil
		},
	}
	testClient := func(t *testing.T, client test.TestingClient) {
		for i := 0; i < 10; i++ {
			start := time.Now()
			resp, err := client.StreamingRequestResponse(context.Background())
			require.NoError(t, err)
			_, err = resp.Recv()
			require.Equal(t, io.EOF, err, "%s. Elapsed: %s", err, time.Since(start))
		}
	}

	kp := maxConnectionAge2GrpcKeepalive(maxAge)
	t.Run("gRPC", func(t *testing.T) {
		testKeepalive(t, false, kp, srv, testClient)
	})
	t.Run("WebSocket", func(t *testing.T) {
		testKeepalive(t, true, kp, srv, testClient)
	})
}

func TestMaxConnectionAgeAndMaxPollDurationRandomizedParallel(t *testing.T) {
	t.Parallel()
	const maxAge = 3 * time.Second
	srv := &test.GrpcTestingServer{
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			time.Sleep(time.Duration(rand.Int63nRange(0, int64(MaxConnectionAge2MaxPollDuration(maxAge)))))
			return nil
		},
	}
	testClient := func(t *testing.T, client test.TestingClient) {
		var wg wait.Group
		defer wg.Wait()
		for i := 0; i < 10; i++ {
			wg.Start(func() {
				for j := 0; j < 10; j++ {
					time.Sleep(time.Duration(rand.Int63nRange(0, int64(maxAge)/10)))
					start := time.Now()
					resp, err := client.StreamingRequestResponse(context.Background())
					require.NoError(t, err)
					_, err = resp.Recv()
					require.Equal(t, io.EOF, err, "%s. Elapsed: %s", err, time.Since(start))
				}
			})
		}
	}

	kp := maxConnectionAge2GrpcKeepalive(maxAge)
	t.Run("gRPC", func(t *testing.T) {
		testKeepalive(t, false, kp, srv, testClient)
	})
	t.Run("WebSocket", func(t *testing.T) {
		testKeepalive(t, true, kp, srv, testClient)
	})
}

func testKeepalive(t *testing.T, websocket bool, kp keepalive.ServerParameters, srv test.TestingServer, f func(*testing.T, test.TestingClient)) {
	t.Parallel()
	l, dial := listenerAndDialer(websocket)
	defer func() {
		assert.NoError(t, l.Close())
	}()
	s := grpc.NewServer(
		grpc.KeepaliveParams(kp),
	)
	defer s.Stop()
	test.RegisterTestingServer(s, srv)
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
