package grpctools

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_grpc"
	"google.golang.org/grpc"
)

type testServerLimiter struct {
	allow bool
}

func (l *testServerLimiter) Allow(ctx context.Context) bool {
	return l.allow
}

func TestServerInterceptors(t *testing.T) {
	ctrl := gomock.NewController(t)
	usHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return struct{}{}, nil
	}
	ssHandler := func(interface{}, grpc.ServerStream) error {
		return nil
	}
	t.Run("It lets the connection through when allowed", func(t *testing.T) {
		limiter := &testServerLimiter{allow: true}

		usi := UnaryServerLimitingInterceptor(limiter)
		_, err := usi(context.Background(), struct{}{}, nil, usHandler)
		require.NoError(t, err)

		ssi := StreamServerLimitingInterceptor(limiter)
		ss := mock_grpc.NewMockServerStream(ctrl)
		ss.EXPECT().Context().Return(context.Background())
		err = ssi(struct{}{}, ss, nil, ssHandler)
		require.NoError(t, err)
	})

	t.Run("It blocks the connection when not allowed", func(t *testing.T) {
		limiter := &testServerLimiter{false}

		usi := UnaryServerLimitingInterceptor(limiter)
		_, err := usi(context.Background(), struct{}{}, nil, usHandler)
		require.Error(t, err)

		ssi := StreamServerLimitingInterceptor(limiter)
		ss := mock_grpc.NewMockServerStream(ctrl)
		ss.EXPECT().Context().Return(context.Background())
		err = ssi(struct{}{}, ss, nil, ssHandler)
		require.Error(t, err)
	})
}
