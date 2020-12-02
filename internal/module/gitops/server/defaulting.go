package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultGitopsPollPeriod               = 20 * time.Second
	defaultGitopsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitopsProjectInfoCacheErrorTTL = 1 * time.Minute
	defaultGitopsMaxManifestFileSize      = 1024 * 1024
	defaultGitopsMaxTotalManifestFileSize = 2 * 1024 * 1024
	defaultGitopsMaxNumberOfPaths         = 100
	defaultGitopsMaxNumberOfFiles         = 1000
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Agent)
	protodefault.NotNil(&config.Agent.Listen)
	protodefault.NotNil(&config.Agent.Gitops)

	gitops := config.Agent.Gitops
	protodefault.Duration(&gitops.PollPeriod, defaultGitopsPollPeriod)
	protodefault.Duration(&gitops.ProjectInfoCacheTtl, defaultGitopsProjectInfoCacheTTL)
	protodefault.Duration(&gitops.ProjectInfoCacheErrorTtl, defaultGitopsProjectInfoCacheErrorTTL)
	protodefault.Uint32(&gitops.MaxManifestFileSize, defaultGitopsMaxManifestFileSize)
	protodefault.Uint32(&gitops.MaxTotalManifestFileSize, defaultGitopsMaxTotalManifestFileSize)
	protodefault.Uint32(&gitops.MaxNumberOfPaths, defaultGitopsMaxNumberOfPaths)
	protodefault.Uint32(&gitops.MaxNumberOfFiles, defaultGitopsMaxNumberOfFiles)
}
