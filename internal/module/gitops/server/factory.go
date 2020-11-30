package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
)

const (
	gitopsSyncCountKnownMetric = "gitops_sync_count"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	projectInfoCacheTtl := config.Config.Agent.Gitops.ProjectInfoCacheTtl.AsDuration()
	projectInfoCacheErrorTtl := config.Config.Agent.Gitops.ProjectInfoCacheErrorTtl.AsDuration()
	m := &module{
		log:        config.Log,
		api:        config.Api,
		gitalyPool: config.Gitaly,
		projectInfoClient: &projectInfoClient{
			GitLabClient:             config.GitLabClient,
			ProjectInfoCacheTtl:      projectInfoCacheTtl,
			ProjectInfoCacheErrorTtl: projectInfoCacheErrorTtl,
			ProjectInfoCache:         cache.New(minDuration(projectInfoCacheTtl, projectInfoCacheErrorTtl)),
		},
		gitopsSyncCount:                config.UsageTracker.RegisterCounter(gitopsSyncCountKnownMetric),
		gitopsPollPeriod:               config.Config.Agent.Gitops.PollPeriod.AsDuration(),
		connectionMaxAge:               config.Config.Agent.Limits.ConnectionMaxAge.AsDuration(),
		maxGitopsManifestFileSize:      int64(config.Config.Agent.Limits.MaxGitopsManifestFileSize),
		maxGitopsTotalManifestFileSize: int64(config.Config.Agent.Limits.MaxGitopsTotalManifestFileSize),
		maxGitopsNumberOfPaths:         config.Config.Agent.Limits.MaxGitopsNumberOfPaths,
		maxGitopsNumberOfFiles:         config.Config.Agent.Limits.MaxGitopsNumberOfFiles,
	}
	rpc.RegisterGitopsServer(config.AgentServer, m)
	return m
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
