package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
)

const (
	defaultGitopsPollPeriod               = 20 * time.Second
	defaultGitopsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitopsProjectInfoCacheErrorTTL = 1 * time.Minute
	defaultGitopsMaxManifestFileSize      = 5 * 1024 * 1024
	defaultGitopsMaxTotalManifestFileSize = 20 * 1024 * 1024
	defaultGitopsMaxNumberOfPaths         = 100
	defaultGitopsMaxNumberOfFiles         = 1000
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	prototool.NotNil(&config.Agent)
	prototool.NotNil(&config.Agent.Listen)
	prototool.NotNil(&config.Agent.Gitops)

	gitops := config.Agent.Gitops
	prototool.Duration(&gitops.PollPeriod, defaultGitopsPollPeriod)
	prototool.Duration(&gitops.ProjectInfoCacheTtl, defaultGitopsProjectInfoCacheTTL)
	prototool.Duration(&gitops.ProjectInfoCacheErrorTtl, defaultGitopsProjectInfoCacheErrorTTL)
	prototool.Uint32(&gitops.MaxManifestFileSize, defaultGitopsMaxManifestFileSize)
	prototool.Uint32(&gitops.MaxTotalManifestFileSize, defaultGitopsMaxTotalManifestFileSize)
	prototool.Uint32(&gitops.MaxNumberOfPaths, defaultGitopsMaxNumberOfPaths)
	prototool.Uint32(&gitops.MaxNumberOfFiles, defaultGitopsMaxNumberOfFiles)
}
