package kas

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Config struct {
	Log                            *zap.Logger
	GitalyPool                     gitaly.PoolInterface
	GitLabClient                   gitlab.ClientInterface
	Registerer                     prometheus.Registerer
	ErrorTracker                   errortracking.Tracker
	AgentConfigurationPollPeriod   time.Duration
	GitopsPollPeriod               time.Duration
	UsageReportingPeriod           time.Duration
	MaxConfigurationFileSize       uint32
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
	gitalyPool                     gitaly.PoolInterface
	gitLabClient                   gitlab.ClientInterface
	errorTracker                   errortracking.Tracker
	agentConfigurationPollPeriod   time.Duration
	gitopsPollPeriod               time.Duration
	usageReportingPeriod           time.Duration
	maxConfigurationFileSize       int64
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
		gitalyPool:                     config.GitalyPool,
		gitLabClient:                   config.GitLabClient,
		errorTracker:                   config.ErrorTracker,
		agentConfigurationPollPeriod:   config.AgentConfigurationPollPeriod,
		gitopsPollPeriod:               config.GitopsPollPeriod,
		usageReportingPeriod:           config.UsageReportingPeriod,
		maxConfigurationFileSize:       int64(config.MaxConfigurationFileSize),
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

func (s *Server) pollImmediateUntil(ctx context.Context, interval time.Duration, condition wait.ConditionFunc) error {
	// this context must only be used here, not inside of condition() - connection should be closed only when idle.
	ageCtx, cancel := context.WithTimeout(ctx, s.connectionMaxAge)
	defer cancel()
	err := retry.PollImmediateUntil(ageCtx, interval, condition)
	if errors.Is(err, wait.ErrWaitTimeout) {
		return nil // all good, ctx is done
	}
	return err
}
