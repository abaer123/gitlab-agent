package redistool

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"go.uber.org/zap"
)

// TokenLimiter is a redis-based rate limiter implementing the algorithm in https://redislabs.com/redis-best-practices/basic-rate-limiting/
type TokenLimiter struct {
	log            *zap.Logger
	redisClient    redis.UniversalClient
	keyPrefix      string
	limitPerMinute uint64
	getToken       func(ctx context.Context) string
}

// NewTokenLimiter returns a new TokenLimiter
func NewTokenLimiter(log *zap.Logger, redisClient redis.UniversalClient, keyPrefix string, limitPerMinute uint64, getToken func(ctx context.Context) string) *TokenLimiter {
	return &TokenLimiter{
		log:            log,
		redisClient:    redisClient,
		keyPrefix:      keyPrefix,
		limitPerMinute: limitPerMinute,
		getToken:       getToken,
	}
}

// Allow consumes one limitable event from the token in the context
func (l *TokenLimiter) Allow(ctx context.Context) bool {
	key := l.buildKey(l.getToken(ctx))

	count, err := l.redisClient.Get(ctx, key).Uint64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			// FIXME: Handle error
			l.log.Error("redistool.TokenLimiter: Error retrieving minute bucket count", zap.Error(err))
			return false
		}
		count = 0
	}
	if count >= l.limitPerMinute {
		l.log.Debug("redistool.TokenLimiter: Rate limit exceeded",
			logz.RedisKey([]byte(key)), logz.U64Count(count), logz.TokenLimit(l.limitPerMinute))
		return false
	}

	// FIXME: Handle errors
	_, err = l.redisClient.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Incr(ctx, key)
		p.Expire(ctx, key, 59*time.Second)
		return nil
	})
	if err != nil {
		l.log.Error("redistool.TokenLimiter: Error wile incrementing token key count", zap.Error(err))
		// FIXME: Handle error
		return false
	}

	return true
}

func (l *TokenLimiter) buildKey(token string) string {
	// We use only the first half of the token as a key. Under the assumption of
	// a randomly generated token of length at least 50, with an alphabet of at least
	//
	// - upper-case characters (26)
	// - lower-case characters (26),
	// - numbers (10),
	//
	// (see https://gitlab.com/gitlab-org/gitlab/blob/master/app/models/clusters/agent_token.rb)
	//
	// we have at least 62^25 different possible token prefixes. Since the token is
	// randomly generated, to obtain the token from this hash, one would have to
	// also guess the second half, and validate it by attempting to log in (kas
	// cannot validate tokens on its own)
	n := len(token) / 2
	tokenHash := sha256.Sum256([]byte(token[:n]))

	currentMinute := time.Now().UTC().Minute()

	var result strings.Builder
	result.WriteString(l.keyPrefix)
	result.WriteByte(':')
	result.Write(tokenHash[:])
	result.WriteByte(':')
	minuteBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(minuteBytes, uint16(currentMinute))
	result.Write(minuteBytes)

	return result.String()
}
