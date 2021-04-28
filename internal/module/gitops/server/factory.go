package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
)

const (
	getObjectsToSynchronizeInitBackoff   = 10 * time.Second
	getObjectsToSynchronizeMaxBackoff    = 5 * time.Minute
	getObjectsToSynchronizeResetDuration = 10 * time.Minute
	getObjectsToSynchronizeBackoffFactor = 2.0
	getObjectsToSynchronizeJitter        = 1.0

	gitopsSyncCountKnownMetric = "gitops_sync_count"
)

type Factory struct {
}

func constructGitOpsPollIntervalHistogram() prometheus.Histogram {
	return prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gitops_poll_interval",
		Help:    "The time between KAS calls to Gitaly to look for gitops updates",
		Buckets: prometheus.LinearBuckets(20, 20, 5), // 5 buckets (20, 40, 60, 80, 100)
	})
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	gitops := config.Config.Agent.Gitops
	projectInfoCacheTtl := gitops.ProjectInfoCacheTtl.AsDuration()
	projectInfoCacheErrorTtl := gitops.ProjectInfoCacheErrorTtl.AsDuration()
	gitOpsPollIntervalHistogram := constructGitOpsPollIntervalHistogram()
	_, err := metric.Register(config.Registerer, gitOpsPollIntervalHistogram)
	if err != nil {
		return nil, err
	}
	m := &module{
		api:        config.Api,
		gitalyPool: config.Gitaly,
		projectInfoClient: &projectInfoClient{
			GitLabClient:             config.GitLabClient,
			ProjectInfoCacheTtl:      projectInfoCacheTtl,
			ProjectInfoCacheErrorTtl: projectInfoCacheErrorTtl,
			ProjectInfoCache:         cache.New(minDuration(projectInfoCacheTtl, projectInfoCacheErrorTtl)),
		},
		syncCount:                   config.UsageTracker.RegisterCounter(gitopsSyncCountKnownMetric),
		gitOpsPollIntervalHistogram: gitOpsPollIntervalHistogram,
		getObjectsBackoff: retry.NewExponentialBackoffFactory(
			getObjectsToSynchronizeInitBackoff,
			getObjectsToSynchronizeMaxBackoff,
			getObjectsToSynchronizeResetDuration,
			getObjectsToSynchronizeBackoffFactor,
			getObjectsToSynchronizeJitter,
		),
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
