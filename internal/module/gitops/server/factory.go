package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/cache"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
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

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	s, err := newServerFromConfig(config)
	if err != nil {
		return nil, err
	}
	rpc.RegisterGitopsServer(config.AgentServer, s)
	return &module{}, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}

func newServerFromConfig(config *modserver.Config) (*server, error) {
	gitOpsPollIntervalHistogram := constructGitOpsPollIntervalHistogram()
	_, err := metric.Register(config.Registerer, gitOpsPollIntervalHistogram)
	if err != nil {
		return nil, err
	}
	gitops := config.Config.Agent.Gitops
	return &server{
		api:        config.Api,
		gitalyPool: config.Gitaly,
		projectInfoClient: &projectInfoClient{
			GitLabClient:     config.GitLabClient,
			ProjectInfoCache: cache.NewWithError(gitops.ProjectInfoCacheTtl.AsDuration(), gitops.ProjectInfoCacheErrorTtl.AsDuration()),
		},
		syncCount:                   config.UsageTracker.RegisterCounter(gitopsSyncCountKnownMetric),
		gitOpsPollIntervalHistogram: gitOpsPollIntervalHistogram,
		getObjectsPollConfig: retry.NewPollConfigFactory(gitops.PollPeriod.AsDuration(), retry.NewExponentialBackoffFactory(
			getObjectsToSynchronizeInitBackoff,
			getObjectsToSynchronizeMaxBackoff,
			getObjectsToSynchronizeResetDuration,
			getObjectsToSynchronizeBackoffFactor,
			getObjectsToSynchronizeJitter,
		)),
		maxManifestFileSize:      int64(gitops.MaxManifestFileSize),
		maxTotalManifestFileSize: int64(gitops.MaxTotalManifestFileSize),
		maxNumberOfPaths:         gitops.MaxNumberOfPaths,
		maxNumberOfFiles:         gitops.MaxNumberOfFiles,
	}, nil
}

func constructGitOpsPollIntervalHistogram() prometheus.Histogram {
	return prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gitops_poll_interval",
		Help:    "The time between kas calls to Gitaly to look for GitOps updates",
		Buckets: prometheus.LinearBuckets(20, 20, 5), // 5 buckets (20, 40, 60, 80, 100)
	})
}
