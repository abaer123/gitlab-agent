package kasapp

import (
	"time"

	agent_configuration_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/server"
	google_profiler_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/google_profiler/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	observability_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultGitLabAddress                = "http://localhost:8080"
	defaultGitLabApiRateLimitRefillRate = 10.0
	defaultGitLabApiRateLimitBucketSize = 50

	defaultAgentListenAddress = "127.0.0.1:8150"

	defaultAgentInfoCacheTTL      = 5 * time.Minute
	defaultAgentInfoCacheErrorTTL = 1 * time.Minute

	defaultAgentLimitsConnectionsPerTokenPerMinute   = 100
	defaultAgentLimitsRedisKeyPrefix                 = "kas:agent_limits"
	defaultAgentLimitsMaxGitopsManifestFileSize      = 1024 * 1024
	defaultAgentLimitsMaxGitopsTotalManifestFileSize = 2 * 1024 * 1024
	defaultAgentLimitsMaxGitopsNumberOfPaths         = 100
	defaultAgentLimitsMaxGitopsNumberOfFiles         = 1000
	defaultAgentLimitsConnectionMaxAge               = 30 * time.Minute

	defaultGitOpsPollPeriod               = 20 * time.Second
	defaultGitOpsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitOpsProjectInfoCacheErrorTTL = 1 * time.Minute

	defaultGitalyGlobalApiRefillRate    = 10.0
	defaultGitalyGlobalApiBucketSize    = 50
	defaultGitalyPerServerApiRate       = 5.0
	defaultGitalyPerServerApiBucketSize = 10

	defaultRedisMaxIdle      = 1
	defaultRedisMaxActive    = 1
	defaultRedisReadTimeout  = 1 * time.Second
	defaultRedisWriteTimeout = 1 * time.Second
	defaultRedisKeepAlive    = 5 * time.Minute
)

var (
	defaulters = []modserver.ApplyDefaults{
		observability_server.ApplyDefaults,
		google_profiler_server.ApplyDefaults,
		agent_configuration_server.ApplyDefaults,
	}
)

func ApplyDefaultsToKasConfigurationFile(cfg *kascfg.ConfigurationFile) {
	protodefault.NotNil(&cfg.Gitlab)
	defaultGitLab(cfg.Gitlab)

	protodefault.NotNil(&cfg.Agent)
	defaultAgent(cfg.Agent)

	protodefault.NotNil(&cfg.Gitaly)
	defaultGitaly(cfg.Gitaly)

	if cfg.Redis != nil {
		defaultRedis(cfg.Redis)
	}
	for _, defaulter := range defaulters {
		defaulter(cfg)
	}
}

func defaultGitLab(g *kascfg.GitLabCF) {
	protodefault.String(&g.Address, defaultGitLabAddress)

	protodefault.NotNil(&g.ApiRateLimit)
	protodefault.Float64(&g.ApiRateLimit.RefillRatePerSecond, defaultGitLabApiRateLimitRefillRate)
	protodefault.Uint32(&g.ApiRateLimit.BucketSize, defaultGitLabApiRateLimitBucketSize)
}

func defaultAgent(a *kascfg.AgentCF) {
	protodefault.NotNil(&a.Listen)
	protodefault.String(&a.Listen.Address, defaultAgentListenAddress)

	protodefault.NotNil(&a.Gitops)
	protodefault.Duration(&a.Gitops.PollPeriod, defaultGitOpsPollPeriod)
	protodefault.Duration(&a.Gitops.ProjectInfoCacheTtl, defaultGitOpsProjectInfoCacheTTL)
	protodefault.Duration(&a.Gitops.ProjectInfoCacheErrorTtl, defaultGitOpsProjectInfoCacheErrorTTL)

	protodefault.Duration(&a.InfoCacheTtl, defaultAgentInfoCacheTTL)
	protodefault.Duration(&a.InfoCacheErrorTtl, defaultAgentInfoCacheErrorTTL)

	protodefault.NotNil(&a.Limits)
	protodefault.Uint32(&a.Limits.ConnectionsPerTokenPerMinute, defaultAgentLimitsConnectionsPerTokenPerMinute)
	protodefault.String(&a.Limits.RedisKeyPrefix, defaultAgentLimitsRedisKeyPrefix)
	protodefault.Uint32(&a.Limits.MaxGitopsManifestFileSize, defaultAgentLimitsMaxGitopsManifestFileSize)
	protodefault.Uint32(&a.Limits.MaxGitopsTotalManifestFileSize, defaultAgentLimitsMaxGitopsTotalManifestFileSize)
	protodefault.Uint32(&a.Limits.MaxGitopsNumberOfPaths, defaultAgentLimitsMaxGitopsNumberOfPaths)
	protodefault.Uint32(&a.Limits.MaxGitopsNumberOfFiles, defaultAgentLimitsMaxGitopsNumberOfFiles)
	protodefault.Duration(&a.Limits.ConnectionMaxAge, defaultAgentLimitsConnectionMaxAge)
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
	protodefault.Uint32(&r.MaxIdle, defaultRedisMaxIdle)
	protodefault.Uint32(&r.MaxActive, defaultRedisMaxActive)
	protodefault.Duration(&r.ReadTimeout, defaultRedisReadTimeout)
	protodefault.Duration(&r.WriteTimeout, defaultRedisWriteTimeout)
	protodefault.Duration(&r.Keepalive, defaultRedisKeepAlive)
}
