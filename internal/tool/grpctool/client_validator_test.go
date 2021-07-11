package grpctool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool/test"
	"google.golang.org/grpc"
)

func TestValidator(t *testing.T) {
	lis := NewDialListener()
	defer lis.Close()
	server := grpc.NewServer()
	defer server.Stop()
	test.RegisterTestingServer(server, &test.GrpcTestingServer{
		UnaryFunc: func(ctx context.Context, request *test.Request) (*test.Response, error) {
			return &test.Response{
				// invalid response because Message is not set
			}, nil
		},
		StreamingFunc: func(server test.Testing_StreamingRequestResponseServer) error {
			return server.Send(&test.Response{
				// invalid response because Message is not set
			})
		},
	})
	go func() {
		assert.NoError(t, server.Serve(lis))
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithInsecure(),
		grpc.WithChainStreamInterceptor(StreamClientValidatingInterceptor),
		grpc.WithChainUnaryInterceptor(UnaryClientValidatingInterceptor),
		grpc.WithContextDialer(lis.DialContext),
	)
	require.NoError(t, err)
	defer conn.Close()
	client := test.NewTestingClient(conn)
	t.Run("invalid unary response", func(t *testing.T) {
		_, err = client.RequestResponse(context.Background(), &test.Request{})
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = invalid server response: invalid Response.Message: value is required")
	})
	t.Run("invalid streaming response", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream, err := client.StreamingRequestResponse(ctx)
		require.NoError(t, err)
		_, err = stream.Recv()
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = invalid server response: invalid Response.Message: value is required")
	})
}
