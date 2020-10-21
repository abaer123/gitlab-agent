package redis

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/FZambia/sentinel"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	// These are defaults that are not currently configurable. To start with,
	// we expose the same configuration options as gitlab-workhorse
	//
	// The default non-Sentinel timeouts are for sanity and should not
	// impact normal operations.

	// Timeout for establishing a new connection (when not using Sentinel)
	defaultConnectTimeout = 30 * time.Second

	// Timeout for connects, reads and writes when using Sentinel. This value is
	// lower than the non-sentinel timeouts because Sentinel docs recommend "in
	// the order of few hundred milliseconds":
	// https://redis.io/topics/sentinel-clients#redis-service-discovery-via-sentinel
	defaultSentinelConnectionTimeouts = 500 * time.Millisecond
	// Timeout before killing Idle connections in the pool
	defaultIdleTimeout = 3 * time.Minute
)

// Pool abstracts a redigo connection pool for testability
type Pool interface {
	GetContext(context.Context) (redigo.Conn, error)
}

// Config is the redis pool configuration
type Config struct {
	URL            *url.URL
	Password       string
	MaxIdle        int32
	MaxActive      int32
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	KeepAlive      time.Duration
	SentinelMaster string
	Sentinels      []*url.URL
}

// NewPool returns a redis pool with the given configuration
func NewPool(cfg *Config) *redigo.Pool {
	if cfg.SentinelMaster != "" {
		return newSentinelPool(cfg)
	}
	return newDefaultPool(cfg)
}

// basePool configures the the common fields for all pool kinds
func basePool(cfg *Config) *redigo.Pool {
	return &redigo.Pool{
		MaxIdle:     int(cfg.MaxIdle),
		MaxActive:   int(cfg.MaxActive),
		Wait:        true,
		IdleTimeout: defaultIdleTimeout,
	}
}

func newDefaultPool(cfg *Config) *redigo.Pool {
	pool := basePool(cfg)
	dopts := newDialOptions(cfg, defaultConnectTimeout, cfg.ReadTimeout, cfg.WriteTimeout)

	pool.Dial = func() (redigo.Conn, error) {
		return dialURL(cfg.URL, dopts...)
	}
	return pool
}

// dialURL encapsulates support for differnt redis URLs
// This is necessary for two reasons:
// 1. redigo itself only supports redis:// URLs via DialURL (otherwise use Dial)
// 2. the sentinel library passes string addresses around, and Sentinel can work with tcp:// and redis://
func dialURL(u *url.URL, dopts ...redigo.DialOption) (redigo.Conn, error) {
	switch u.Scheme {
	case "unix":
		return redigo.Dial("unix", u.Path, dopts...)
	case "redis", "rediss":
		return redigo.DialURL(u.String(), dopts...)
	case "":
		return redigo.Dial("tcp", u.Host, dopts...)
	default:
		return redigo.Dial(u.Scheme, u.Host, dopts...)
	}
}

// dialAddr is like dialURL but for sentinel
func dialAddr(addr string, dopts ...redigo.DialOption) (redigo.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		// This should not be possible because the configuration expects url.URL
		return nil, fmt.Errorf("failed to parse redis URL: %v", err)
	}
	return dialURL(u, dopts...)
}

func newDialOptions(cfg *Config, connectTimeout, readTimeout, writeTimeout time.Duration) []redigo.DialOption {
	dopts := []redigo.DialOption{
		redigo.DialConnectTimeout(connectTimeout),
		redigo.DialReadTimeout(readTimeout),
		redigo.DialWriteTimeout(writeTimeout),
		redigo.DialKeepAlive(cfg.KeepAlive),
	}
	if cfg.Password != "" {
		dopts = append(dopts, redigo.DialPassword(cfg.Password))
	}
	return dopts
}

func newSentinelPool(cfg *Config) *redigo.Pool {
	sentinelDopts := newDialOptions(cfg, defaultSentinelConnectionTimeouts, defaultSentinelConnectionTimeouts, defaultSentinelConnectionTimeouts)
	sentinelAddrs := make([]string, 0, len(cfg.Sentinels))
	for _, u := range cfg.Sentinels {
		sentinelAddrs = append(sentinelAddrs, u.String())
	}
	sntnl := &sentinel.Sentinel{
		Addrs:      sentinelAddrs,
		MasterName: cfg.SentinelMaster,
		Dial: func(addr string) (redigo.Conn, error) {
			return dialAddr(addr, sentinelDopts...)
		},
	}
	pool := basePool(cfg)
	poolDopts := newDialOptions(cfg, defaultConnectTimeout, cfg.ReadTimeout, cfg.WriteTimeout)
	pool.Dial = func() (redigo.Conn, error) {
		addr, err := sntnl.MasterAddr()
		if err != nil {
			return nil, err
		}
		return dialAddr(addr, poolDopts...)
	}
	pool.TestOnBorrow = func(c redigo.Conn, t time.Time) error {
		if !sentinel.TestRole(c, "master") {
			return errors.New("role check failed")
		}
		return nil
	}
	return pool
}
