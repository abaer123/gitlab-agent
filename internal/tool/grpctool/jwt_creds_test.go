package grpctool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ credentials.PerRPCCredentials = &JwtCredentials{}
)

const (
	secret   = "dfjnfkadskfadsnfkjadsgkasdbg"
	audience = "fasfadsf"
	issuer   = "cbcxvbvxbxb"
)

func TestJwtCredentialsProducesValidToken(t *testing.T) {
	c := &JwtCredentials{
		Secret:   []byte(secret),
		Audience: audience,
		Issuer:   issuer,
		Insecure: true,
	}
	auther := JWTAuther{
		Secret:   []byte(secret),
		Audience: audience,
		Issuer:   issuer,
	}
	listener := NewDialListener()

	srv := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			auther.StreamServerInterceptor,
		),
		grpc.ChainUnaryInterceptor(
			auther.UnaryServerInterceptor,
		),
	)
	test.RegisterTestingServer(srv, &authTestingServer{})
	var wg wait.Group
	defer wg.Wait()
	defer srv.Stop()
	wg.Start(func() {
		assert.NoError(t, srv.Serve(listener))
	})
	conn, err := grpc.DialContext(context.Background(), "pipe",
		grpc.WithContextDialer(listener.DialContext),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(c),
	)
	require.NoError(t, err)
	client := test.NewTestingClient(conn)
	_, err = client.RequestResponse(context.Background(), &test.Request{})
	require.NoError(t, err)
	stream, err := client.StreamingRequestResponse(context.Background())
	require.NoError(t, err)
	_, err = stream.Recv()
	require.NoError(t, err)
}

type authTestingServer struct {
	test.UnimplementedTestingServer
}

func (t *authTestingServer) RequestResponse(ctx context.Context, request *test.Request) (*test.Response, error) {
	return &test.Response{
		Message: &test.Response_Scalar{Scalar: 123},
	}, nil
}

func (t *authTestingServer) StreamingRequestResponse(server test.Testing_StreamingRequestResponseServer) error {
	return server.Send(&test.Response{
		Message: &test.Response_Scalar{Scalar: 123},
	})
}
