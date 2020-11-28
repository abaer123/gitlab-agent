package kas

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/metric"
	"go.uber.org/zap"
)

const (
	gitopsSyncCountKnownMetric = "gitops_sync_count"
)

type Config struct {
	Log                            *zap.Logger
	Api                            modserver.API
	GitalyPool                     gitaly.PoolInterface
	GitLabClient                   gitlab.ClientInterface
	Registerer                     prometheus.Registerer
	UsageTracker                   usage_metrics.UsageTrackerRegisterer
	GitopsPollPeriod               time.Duration
	MaxGitopsManifestFileSize      uint32
	MaxGitopsTotalManifestFileSize uint32
	MaxGitopsNumberOfPaths         uint32
	MaxGitopsNumberOfFiles         uint32
	ConnectionMaxAge               time.Duration
}

type Server struct {
	log                            *zap.Logger
	api                            modserver.API
	gitalyPool                     gitaly.PoolInterface
	gitLabClient                   gitlab.ClientInterface
	gitopsSyncCount                usage_metrics.Counter
	gitopsPollPeriod               time.Duration
	maxGitopsManifestFileSize      int64
	maxGitopsTotalManifestFileSize int64
	maxGitopsNumberOfPaths         uint32
	maxGitopsNumberOfFiles         uint32
	connectionMaxAge               time.Duration
}

func NewServer(config Config) (*Server, func(), error) {
	toRegister := []prometheus.Collector{
		// TODO add actual metrics
	}
	cleanup, err := metric.Register(config.Registerer, toRegister...)
	if err != nil {
		return nil, nil, err
	}
	s := &Server{
		log:                            config.Log,
		api:                            config.Api,
		gitalyPool:                     config.GitalyPool,
		gitLabClient:                   config.GitLabClient,
		gitopsSyncCount:                config.UsageTracker.RegisterCounter(gitopsSyncCountKnownMetric),
		gitopsPollPeriod:               config.GitopsPollPeriod,
		maxGitopsManifestFileSize:      int64(config.MaxGitopsManifestFileSize),
		maxGitopsTotalManifestFileSize: int64(config.MaxGitopsTotalManifestFileSize),
		maxGitopsNumberOfPaths:         config.MaxGitopsNumberOfPaths,
		maxGitopsNumberOfFiles:         config.MaxGitopsNumberOfFiles,
		connectionMaxAge:               config.ConnectionMaxAge,
	}
	return s, cleanup, nil
}
