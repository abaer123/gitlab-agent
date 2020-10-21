package redis

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"gitlab.com/gitlab-org/labkit/log"
)

// TokenLimiter is a redis-based rate limiter implementing the algorithm in https://redislabs.com/redis-best-practices/basic-rate-limiting/
type TokenLimiter struct {
	redisPool      Pool
	keyPrefix      string
	limitPerMinute uint64
	getToken       func(ctx context.Context) string
}

// NewTokenLimiter returns a new TokenLimiter
func NewTokenLimiter(redisPool Pool, keyPrefix string, limitPerMinute uint64, getToken func(ctx context.Context) string) *TokenLimiter {
	return &TokenLimiter{
		redisPool:      redisPool,
		keyPrefix:      keyPrefix,
		limitPerMinute: limitPerMinute,
		getToken:       getToken,
	}
}

// Allow consumes one limitable event from the token in the context
func (l *TokenLimiter) Allow(ctx context.Context) bool {
	token := l.getToken(ctx)
	conn, err := l.redisPool.GetContext(ctx)
	if err != nil {
		// FIXME: Handle error.
		log.WithError(err).Error("redis.TokenLimiter: Error connecting to redis")
		return false
	}
	defer conn.Close() // nolint: errcheck

	key := l.buildKey(token)

	count, err := redigo.Uint64(conn.Do("GET", key))
	if err != nil {
		if err != redigo.ErrNil {
			// FIXME: Handle error
			log.WithError(err).Error("redis.TokenLimiter: Error retrieving minute bucket count")
			return false
		}
		count = 0
	}
	if count >= l.limitPerMinute {
		log.WithFields(log.Fields{
			"key":   string(key),
			"count": count,
			"limit": l.limitPerMinute,
		}).Warn("redis.TokenLimiter: Rate limit exceeded")
		return false
	}

	// FIXME: Handle errors
	err = conn.Send("MULTI")
	if err != nil {
		return false
	}
	err = conn.Send("INCR", key)
	if err != nil {
		return false
	}
	err = conn.Send("EXPIRE", key, 59)
	if err != nil {
		return false
	}
	_, err = conn.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("redis.TokenLimiter: Error wile incrementing token key count")
		// FIXME: Handle error
		return false
	}

	return true
}

func (l *TokenLimiter) buildKey(token string) []byte {
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
	result := append([]byte(l.keyPrefix), ':')
	result = append(result, tokenHash[:]...)
	result = append(result, ':')
	minuteBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(minuteBytes, uint16(currentMinute))
	result = append(result, minuteBytes...)

	return result
}
