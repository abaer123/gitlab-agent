package redis

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_redis"
)

func TestTokenLimiter(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPool := mock_redis.NewMockPool(ctrl)

	limitPerMinute := uint64(1)
	limiter := NewTokenLimiter(mockPool, "key_prefix", limitPerMinute, func(ctx context.Context) string { return string(apiutil.AgentTokenFromContext(ctx)) })
	token := api.AgentToken("0123456789")
	meta := &api.AgentMeta{Token: token}
	ctx := apiutil.InjectAgentMeta(context.Background(), meta)
	key := limiter.buildKey(string(token))

	require.True(t, strings.HasPrefix(string(key), "key_prefix:"), "Key should have the key prefix")

	expectWithCount(ctrl, mockPool, ctx, key, limitPerMinute-1)
	require.True(t, limiter.Allow(ctx), "Allow when no token has been consumed")

	expectWithCount(ctrl, mockPool, ctx, key, limitPerMinute)
	require.False(t, limiter.Allow(ctx), "Do not allow when a token has been consumed")

	mockPool.EXPECT().GetContext(ctx).Return(nil, fmt.Errorf("test connection error"))
	require.False(t, limiter.Allow(ctx), "Do not allow when there is a connection error")
}

func expectWithCount(ctrl *gomock.Controller, mockPool *mock_redis.MockPool, ctx context.Context, key []byte, count uint64) {
	mockConn := mock_redis.NewMockConn(ctrl)
	gomock.InOrder(
		mockPool.EXPECT().GetContext(ctx).Return(mockConn, nil),
		mockConn.EXPECT().Do("GET", key).Return(interface{}(int64(count)), nil),
		// these steps depend on the count, hence the MaxTimes(1)
		mockConn.EXPECT().Send("MULTI").Return(nil).MaxTimes(1),
		mockConn.EXPECT().Send("INCR", key).Return(nil).MaxTimes(1),
		mockConn.EXPECT().Send("EXPIRE", key).Return(nil).MaxTimes(1),
		mockConn.EXPECT().Do("EXEC").Return(gomock.Any(), nil).MaxTimes(1),
		// this step is mandatory and should happen last
		mockConn.EXPECT().Close().Times(1),
	)
}
