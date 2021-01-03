package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
)

const (
	gitopsSyncCountKnownMetric = "gitops_sync_count"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	gitops := config.Config.Agent.Gitops
	projectInfoCacheTtl := gitops.ProjectInfoCacheTtl.AsDuration()
	projectInfoCacheErrorTtl := gitops.ProjectInfoCacheErrorTtl.AsDuration()
	m := &module{
		api:        config.Api,
		gitalyPool: config.Gitaly,
		projectInfoClient: &projectInfoClient{
			GitLabClient:             config.GitLabClient,
			ProjectInfoCacheTtl:      projectInfoCacheTtl,
			ProjectInfoCacheErrorTtl: projectInfoCacheErrorTtl,
			ProjectInfoCache:         cache.New(minDuration(projectInfoCacheTtl, projectInfoCacheErrorTtl)),
		},
		syncCount:                config.UsageTracker.RegisterCounter(gitopsSyncCountKnownMetric),
		pollPeriod:               gitops.PollPeriod.AsDuration(),
		maxConnectionAge:         config.Config.Agent.Listen.MaxConnectionAge.AsDuration(),
		maxManifestFileSize:      int64(gitops.MaxManifestFileSize),
		maxTotalManifestFileSize: int64(gitops.MaxTotalManifestFileSize),
		maxNumberOfPaths:         gitops.MaxNumberOfPaths,
		maxNumberOfFiles:         gitops.MaxNumberOfFiles,
	}
	rpc.RegisterGitopsServer(config.AgentServer, m)
	return m, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
