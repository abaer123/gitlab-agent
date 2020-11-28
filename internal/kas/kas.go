package kas

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/metric"
	"go.uber.org/zap"
)

type Config struct {
	Log                            *zap.Logger
	Api                            modserver.API
	GitalyPool                     gitaly.PoolInterface
	GitLabClient                   gitlab.ClientInterface
	Registerer                     prometheus.Registerer
	GitopsPollPeriod               time.Duration
	UsageReportingPeriod           time.Duration
	MaxGitopsManifestFileSize      uint32
	MaxGitopsTotalManifestFileSize uint32
	MaxGitopsNumberOfPaths         uint32
	MaxGitopsNumberOfFiles         uint32
	ConnectionMaxAge               time.Duration
}

type Server struct {
	// usageMetrics must be the very first field to ensure 64-bit alignment.
	// See https://github.com/golang/go/blob/95df156e6ac53f98efd6c57e4586c1dfb43066dd/src/sync/atomic/doc.go#L46-L54
	usageMetrics                   usageMetrics
	log                            *zap.Logger
	api                            modserver.API
	gitalyPool                     gitaly.PoolInterface
	gitLabClient                   gitlab.ClientInterface
	gitopsPollPeriod               time.Duration
	usageReportingPeriod           time.Duration
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
		gitopsPollPeriod:               config.GitopsPollPeriod,
		usageReportingPeriod:           config.UsageReportingPeriod,
		maxGitopsManifestFileSize:      int64(config.MaxGitopsManifestFileSize),
		maxGitopsTotalManifestFileSize: int64(config.MaxGitopsTotalManifestFileSize),
		maxGitopsNumberOfPaths:         config.MaxGitopsNumberOfPaths,
		maxGitopsNumberOfFiles:         config.MaxGitopsNumberOfFiles,
		connectionMaxAge:               config.ConnectionMaxAge,
	}
	return s, cleanup, nil
}

func (s *Server) Run(ctx context.Context) {
	s.sendUsage(ctx)
}
