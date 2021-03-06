package kasapp

import (
	"time"

	agent_configuration_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/server"
	gitops_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/server"
	google_profiler_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/google_profiler/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	observability_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/server"
	usage_metrics_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultGitLabApiRateLimitRefillRate = 10.0
	defaultGitLabApiRateLimitBucketSize = 50

	defaultAgentInfoCacheTTL         = 5 * time.Minute
	defaultAgentInfoCacheErrorTTL    = 1 * time.Minute
	defaultAgentRedisConnInfoTTL     = 5 * time.Minute
	defaultAgentRedisConnInfoRefresh = 4 * time.Minute
	defaultAgentRedisConnInfoGC      = 10 * time.Minute

	defaultAgentListenAddress                      = "127.0.0.1:8150"
	defaultAgentListenConnectionsPerTokenPerMinute = 100
	defaultAgentListenMaxConnectionAge             = 30 * time.Minute

	defaultGitalyGlobalApiRefillRate    = 10.0
	defaultGitalyGlobalApiBucketSize    = 50
	defaultGitalyPerServerApiRate       = 5.0
	defaultGitalyPerServerApiBucketSize = 10

	defaultRedisPoolSize     = 5
	defaultRedisDialTimeout  = 5 * time.Second
	defaultRedisReadTimeout  = 1 * time.Second
	defaultRedisWriteTimeout = 1 * time.Second
	defaultRedisIdleTimeout  = 5 * time.Minute
	defaultRedisKeyPrefix    = "gitlab-kas"

	defaultApiListenAddress          = "127.0.0.1:8153"
	defaultApiListenMaxConnectionAge = 30 * time.Minute
)

var (
	defaulters = []modserver.ApplyDefaults{
		observability_server.ApplyDefaults,
		google_profiler_server.ApplyDefaults,
		agent_configuration_server.ApplyDefaults,
		usage_metrics_server.ApplyDefaults,
		gitops_server.ApplyDefaults,
	}
)

func ApplyDefaultsToKasConfigurationFile(cfg *kascfg.ConfigurationFile) {
	protodefault.NotNil(&cfg.Gitlab)
	defaultGitLab(cfg.Gitlab)

	protodefault.NotNil(&cfg.Agent)
	defaultAgent(cfg.Agent)

	protodefault.NotNil(&cfg.Gitaly)
	defaultGitaly(cfg.Gitaly)

	// TODO this should become required
	if cfg.Redis != nil {
		defaultRedis(cfg.Redis)
	}

	// TODO this should become required
	if cfg.Api != nil {
		defaultApi(cfg.Api)
	}

	for _, defaulter := range defaulters {
		defaulter(cfg)
	}
}

func defaultApi(api *kascfg.ApiCF) {
	protodefault.NotNil(&api.Listen)
	protodefault.String(&api.Listen.Address, defaultApiListenAddress)
	protodefault.Duration(&api.Listen.MaxConnectionAge, defaultApiListenMaxConnectionAge)
}

func defaultGitLab(g *kascfg.GitLabCF) {
	protodefault.NotNil(&g.ApiRateLimit)
	protodefault.Float64(&g.ApiRateLimit.RefillRatePerSecond, defaultGitLabApiRateLimitRefillRate)
	protodefault.Uint32(&g.ApiRateLimit.BucketSize, defaultGitLabApiRateLimitBucketSize)
}

func defaultAgent(a *kascfg.AgentCF) {
	protodefault.NotNil(&a.Listen)
	protodefault.String(&a.Listen.Address, defaultAgentListenAddress)
	protodefault.Uint32(&a.Listen.ConnectionsPerTokenPerMinute, defaultAgentListenConnectionsPerTokenPerMinute)
	protodefault.Duration(&a.Listen.MaxConnectionAge, defaultAgentListenMaxConnectionAge)

	protodefault.Duration(&a.InfoCacheTtl, defaultAgentInfoCacheTTL)
	protodefault.Duration(&a.InfoCacheErrorTtl, defaultAgentInfoCacheErrorTTL)
	protodefault.Duration(&a.RedisConnInfoTtl, defaultAgentRedisConnInfoTTL)
	protodefault.Duration(&a.RedisConnInfoRefresh, defaultAgentRedisConnInfoRefresh)
	protodefault.Duration(&a.RedisConnInfoGc, defaultAgentRedisConnInfoGC)
}

func defaultGitaly(g *kascfg.GitalyCF) {
	protodefault.NotNil(&g.GlobalApiRateLimit)
	protodefault.Float64(&g.GlobalApiRateLimit.RefillRatePerSecond, defaultGitalyGlobalApiRefillRate)
	protodefault.Uint32(&g.GlobalApiRateLimit.BucketSize, defaultGitalyGlobalApiBucketSize)

	protodefault.NotNil(&g.PerServerApiRateLimit)
	protodefault.Float64(&g.PerServerApiRateLimit.RefillRatePerSecond, defaultGitalyPerServerApiRate)
	protodefault.Uint32(&g.PerServerApiRateLimit.BucketSize, defaultGitalyPerServerApiBucketSize)
}

func defaultRedis(r *kascfg.RedisCF) {
	protodefault.Uint32(&r.PoolSize, defaultRedisPoolSize)
	protodefault.Duration(&r.DialTimeout, defaultRedisDialTimeout)
	protodefault.Duration(&r.ReadTimeout, defaultRedisReadTimeout)
	protodefault.Duration(&r.WriteTimeout, defaultRedisWriteTimeout)
	protodefault.Duration(&r.IdleTimeout, defaultRedisIdleTimeout)
	protodefault.String(&r.KeyPrefix, defaultRedisKeyPrefix)
}
