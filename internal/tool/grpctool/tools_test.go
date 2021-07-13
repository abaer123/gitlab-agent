package grpctool_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/labkit/correlation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type customKey int

const (
	key customKey = iota
)

func TestJoinContexts(t *testing.T) {
	t.Run("inherits values", func(t *testing.T) {
		mainCtx := context.WithValue(context.Background(), key, 2)
		aux := context.Background()

		augmented, err := grpctool.JoinContexts(aux)(mainCtx)
		require.NoError(t, err)
		assert.Equal(t, 2, augmented.Value(key))
	})
	t.Run("propagates cancel from main", func(t *testing.T) {
		mainCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		aux := context.Background()

		augmented, err := grpctool.JoinContexts(aux)(mainCtx)
		require.NoError(t, err)

		cancel()
		<-augmented.Done() // does not block
		assert.Equal(t, context.Canceled, augmented.Err())
	})
	t.Run("propagates cancel from aux", func(t *testing.T) {
		mainCtx := context.Background()
		aux, cancel := context.WithCancel(context.Background())
		defer cancel()

		augmented, err := grpctool.JoinContexts(aux)(mainCtx)
		require.NoError(t, err)

		cancel()
		<-augmented.Done() // does not block
		assert.Equal(t, context.Canceled, augmented.Err())
	})
}

func TestRequestCanceled(t *testing.T) {
	t.Run("context errors", func(t *testing.T) {
		assert.True(t, grpctool.RequestCanceled(context.Canceled))
		assert.True(t, grpctool.RequestCanceled(context.DeadlineExceeded))
		assert.False(t, grpctool.RequestCanceled(io.EOF))
	})
	t.Run("wrapped context errors", func(t *testing.T) {
		assert.True(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", context.Canceled)))
		assert.True(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", context.DeadlineExceeded)))
		assert.False(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", io.EOF)))
	})
	t.Run("gRPC errors", func(t *testing.T) {
		assert.True(t, grpctool.RequestCanceled(status.Error(codes.Canceled, "bla")))
		assert.True(t, grpctool.RequestCanceled(status.Error(codes.DeadlineExceeded, "bla")))
		assert.False(t, grpctool.RequestCanceled(status.Error(codes.Unavailable, "bla")))
	})
	t.Run("wrapped gRPC errors", func(t *testing.T) {
		assert.True(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.Canceled, "bla"))))
		assert.True(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.DeadlineExceeded, "bla"))))
		assert.False(t, grpctool.RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.Unavailable, "bla"))))
	})
}

const metadataCorrelatorKey = "X-GitLab-Correlation-ID"

func TestMaybeWrapWithCorrelationId(t *testing.T) {
	t.Run("header error", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(nil, errors.New("header error"))
		err := errors.New("boom")
		wrappedErr := grpctool.MaybeWrapWithCorrelationId(err, stream)
		assert.Equal(t, err, wrappedErr)
	})
	t.Run("id present", func(t *testing.T) {
		id := correlation.SafeRandomID()
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.Pairs(metadataCorrelatorKey, id), nil)
		err := errors.New("boom")
		wrappedErr := grpctool.MaybeWrapWithCorrelationId(err, stream)
		var errCorrelation errz.CorrelationError
		require.True(t, errors.As(wrappedErr, &errCorrelation))
		assert.Equal(t, id, errCorrelation.CorrelationId)
		assert.Equal(t, err, errCorrelation.Err)
	})
	t.Run("empty id", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.Pairs(metadataCorrelatorKey, ""), nil)
		err := errors.New("boom")
		wrappedErr := grpctool.MaybeWrapWithCorrelationId(err, stream)
		assert.Equal(t, err, wrappedErr)
	})
	t.Run("id missing", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.MD{}, nil)
		err := errors.New("boom")
		wrappedErr := grpctool.MaybeWrapWithCorrelationId(err, stream)
		assert.Equal(t, err, wrappedErr)
	})
}

func TestDeferMaybeWrapWithCorrelationId(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		var wrappedErr error
		grpctool.DeferMaybeWrapWithCorrelationId(&wrappedErr, stream)
		assert.NoError(t, wrappedErr)
	})
	t.Run("header error", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(nil, errors.New("header error"))
		err := errors.New("boom")
		wrappedErr := err
		grpctool.DeferMaybeWrapWithCorrelationId(&wrappedErr, stream)
		assert.Equal(t, err, wrappedErr)
	})
	t.Run("id present", func(t *testing.T) {
		id := correlation.SafeRandomID()
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.Pairs(metadataCorrelatorKey, id), nil)
		err := errors.New("boom")
		wrappedErr := err
		grpctool.DeferMaybeWrapWithCorrelationId(&wrappedErr, stream)
		var errCorrelation errz.CorrelationError
		require.True(t, errors.As(wrappedErr, &errCorrelation))
		assert.Equal(t, id, errCorrelation.CorrelationId)
		assert.Equal(t, err, errCorrelation.Err)
	})
	t.Run("empty id", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.Pairs(metadataCorrelatorKey, ""), nil)
		err := errors.New("boom")
		wrappedErr := err
		grpctool.DeferMaybeWrapWithCorrelationId(&wrappedErr, stream)
		assert.Equal(t, err, wrappedErr)
	})
	t.Run("id missing", func(t *testing.T) {
		stream := mock_rpc.NewMockClientStream(gomock.NewController(t))
		stream.EXPECT().Header().Return(metadata.MD{}, nil)
		err := errors.New("boom")
		wrappedErr := err
		grpctool.DeferMaybeWrapWithCorrelationId(&wrappedErr, stream)
		assert.Equal(t, err, wrappedErr)
	})
}
