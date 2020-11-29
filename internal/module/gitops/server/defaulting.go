package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultAgentLimitsMaxGitopsManifestFileSize      = 1024 * 1024
	defaultAgentLimitsMaxGitopsTotalManifestFileSize = 2 * 1024 * 1024
	defaultAgentLimitsMaxGitopsNumberOfPaths         = 100
	defaultAgentLimitsMaxGitopsNumberOfFiles         = 1000

	defaultGitOpsPollPeriod               = 20 * time.Second
	defaultGitOpsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitOpsProjectInfoCacheErrorTTL = 1 * time.Minute
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Agent)

	a := config.Agent
	protodefault.NotNil(&a.Gitops)
	protodefault.Duration(&a.Gitops.PollPeriod, defaultGitOpsPollPeriod)
	protodefault.Duration(&a.Gitops.ProjectInfoCacheTtl, defaultGitOpsProjectInfoCacheTTL)
	protodefault.Duration(&a.Gitops.ProjectInfoCacheErrorTtl, defaultGitOpsProjectInfoCacheErrorTTL)

	protodefault.NotNil(&a.Limits)
	protodefault.Uint32(&a.Limits.MaxGitopsManifestFileSize, defaultAgentLimitsMaxGitopsManifestFileSize)
	protodefault.Uint32(&a.Limits.MaxGitopsTotalManifestFileSize, defaultAgentLimitsMaxGitopsTotalManifestFileSize)
	protodefault.Uint32(&a.Limits.MaxGitopsNumberOfPaths, defaultAgentLimitsMaxGitopsNumberOfPaths)
	protodefault.Uint32(&a.Limits.MaxGitopsNumberOfFiles, defaultAgentLimitsMaxGitopsNumberOfFiles)
}
