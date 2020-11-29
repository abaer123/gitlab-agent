package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

const (
	gitopsSyncCountKnownMetric = "gitops_sync_count"
)

type Factory struct {
	GitLabClient gitlab.ClientInterface
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	m := &module{
		log:                            config.Log,
		api:                            config.Api,
		gitalyPool:                     config.Gitaly,
		gitLabClient:                   f.GitLabClient,
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
