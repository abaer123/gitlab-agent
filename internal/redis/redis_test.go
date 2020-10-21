package redis

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPoolDefault(t *testing.T) {
	u, err := url.Parse("unix:///var/redis/redis.sock")
	require.NoError(t, err, "URL should be valid")
	cfg := &Config{
		URL: u,
	}
	pool := NewPool(cfg)
	require.Nil(t, pool.TestOnBorrow, "Sentinel should not be enabled by default")
}

func TestNewPoolWithSentinel(t *testing.T) {
	u, err := url.Parse("tcp://localhost:6379")
	require.NoError(t, err, "Invalid URL")
	sentinels := []*url.URL{u, u}
	cfg := &Config{
		SentinelMaster: "master-name",
		Sentinels:      sentinels,
	}
	pool := NewPool(cfg)
	require.NotNil(t, pool.TestOnBorrow, "Sentinel requires a TestOnBorrow check")
}
