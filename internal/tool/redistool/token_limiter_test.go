package redistool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"go.uber.org/zap/zaptest"
)

const (
	token api.AgentToken = "0123456789"
)

func TestTokenLimiterHappyPath(t *testing.T) {
	mock, limiter, ctx, key := setup(t)

	mock.ExpectGet(key).SetVal("0")
	mock.ExpectTxPipeline()
	mock.ExpectIncr(key).SetVal(1)
	mock.ExpectExpire(key, 59*time.Second).SetVal(true)
	mock.ExpectTxPipelineExec()

	require.True(t, limiter.Allow(ctx), "Allow when no token has been consumed")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenLimiterOverLimit(t *testing.T) {
	mock, limiter, ctx, key := setup(t)

	mock.ExpectGet(key).SetVal("1")

	require.False(t, limiter.Allow(ctx), "Do not allow when a token has been consumed")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenLimiterNotAllowedWhenGetError(t *testing.T) {
	mock, limiter, ctx, key := setup(t)
	mock.ExpectGet(key).SetErr(errors.New("test connection error"))

	require.False(t, limiter.Allow(ctx), "Do not allow when there is a connection error")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenLimiterNotAllowedWhenIncrError(t *testing.T) {
	mock, limiter, ctx, key := setup(t)

	mock.ExpectGet(key).SetVal("0")
	mock.ExpectTxPipeline()
	mock.ExpectIncr(key).SetVal(1)
	mock.ExpectExpire(key, 59*time.Second).SetErr(errors.New("test connection error"))

	require.False(t, limiter.Allow(ctx), "Do not allow when there is a connection error")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func setup(t *testing.T) (redismock.ClientMock, *TokenLimiter, context.Context, string) {
	client, mock := redismock.NewClientMock()
	mock.MatchExpectationsInOrder(true)
	limiter := NewTokenLimiter(zaptest.NewLogger(t), client, "key_prefix", 1, tokenFromContext)
	ctx := api.InjectAgentMD(context.Background(), &api.AgentMD{Token: token})
	key := limiter.buildKey(string(token))
	return mock, limiter, ctx, key
}

func tokenFromContext(ctx context.Context) string {
	return string(api.AgentTokenFromContext(ctx))
}
