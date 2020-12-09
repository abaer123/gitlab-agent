package modserver

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ApplyDefaults is a signature of a public function, exposed by modules to perform defaulting.
// The function should be called ApplyDefaults.
type ApplyDefaults func(*kascfg.ConfigurationFile)

// Config holds configuration for a Module.
type Config struct {
	Log          *zap.Logger
	Api          API
	Config       *kascfg.ConfigurationFile
	GitLabClient gitlab.ClientInterface
	// Registerer allows to register metrics.
	// Metrics should be registered in Run and unregistered before Run returns.
	Registerer   prometheus.Registerer
	UsageTracker usage_metrics.UsageTrackerRegisterer
	// AgentServer is the gRPC server agentk is talking to.
	// This can be used to add endpoints in Factory.New.
	AgentServer *grpc.Server
	Gitaly      gitaly.PoolInterface
	// KasName is a string "gitlab-kas". Can be used as a user agent, server name, service name, etc.
	KasName string
	// Version is gitlab-kas version.
	Version string
	// CommitId is gitlab-kas commit.
	CommitId string
}

// API provides the API for the module to use.
type API interface {
	errortracking.Tracker
	// GetAgentInfo encapsulates error checking logic.
	// The signature is not conventional on purpose - the caller is not supposed to inspect the error,
	// but instead return it if the bool is true. If the bool is false, AgentInfo is non-nil.
	GetAgentInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken, noErrorOnUnknownError bool) (*api.AgentInfo, error, bool /* return the error? */)
	// PollImmediateUntil should be used by the top-level polling, so that it can be gracefully interrupted
	// by the server when necessary.
	PollImmediateUntil(ctx context.Context, interval, maxConnectionAge time.Duration, condition ConditionFunc) error
	HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error)
	HandleSendError(log *zap.Logger, msg string, err error) error
	LogAndCapture(ctx context.Context, log *zap.Logger, msg string, err error)
}

type Factory interface {
	// New creates a new instance of a Module.
	New(*Config) Module
}

type Module interface {
	// Run starts the module.
	// Run can block until the context is canceled or exit with nil if there is nothing to do.
	Run(context.Context) error
	// Name returns module's name.
	Name() string
}

// ConditionFunc returns true if the condition is satisfied, or an error
// if the loop should be aborted.
type ConditionFunc func() (done bool, err error)
