package modserver

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// RoutingHopPrefix is a metadata key prefix that is used for metadata keys that should be consumed by
	// the gateway kas instances and not passed along to agentk.
	RoutingHopPrefix = "kas-hop-"
	// RoutingAgentIdMetadataKey is used to pass destination agent id in request metadata
	// from the routing kas instance, that is handling the incoming request, to the gateway kas instance,
	// that is forwarding the request to an agentk.
	RoutingAgentIdMetadataKey = RoutingHopPrefix + "routing-agent-id"
)

// ApplyDefaults is a signature of a public function, exposed by modules to perform defaulting.
// The function should be called ApplyDefaults.
type ApplyDefaults func(*kascfg.ConfigurationFile)

// Config holds configuration for a Module.
type Config struct {
	// Log can be used for logging from the module.
	// It should not be used for logging from gRPC API methods. Use grpctool.LoggerFromContext(ctx) instead.
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
	// Request handlers can obtain the per-request logger using grpctool.LoggerFromContext(requestContext).
	AgentServer *grpc.Server
	// ApiServer is the gRPC server GitLab is talking to.
	// This can be used to add endpoints in Factory.New.
	// Request handlers can obtain the per-request logger using grpctool.LoggerFromContext(requestContext).
	ApiServer *grpc.Server
	// RegisterAgentApi allows to register a gRPC API endpoint that kas proxies to agentk.
	RegisterAgentApi func(*grpc.ServiceDesc)
	// AgentConn is a gRPC connection that can be used to send requests to an agentk instance.
	// Agent Id must be specified in the request metadata in RoutingAgentIdMetadataKey field.
	AgentConn grpc.ClientConnInterface
	Gitaly    gitaly.PoolInterface
	// KasName is a string "gitlab-kas". Can be used as a user agent, server name, service name, etc.
	KasName string
	// Version is gitlab-kas version.
	Version string
	// CommitId is gitlab-kas commit sha.
	CommitId string
}

// API provides the API for the module to use.
type API interface {
	modshared.API
	// GetAgentInfo encapsulates error checking logic.
	// The signature is not conventional on purpose - the caller is not supposed to inspect the error,
	// but instead return it if the bool is true. If the bool is false, AgentInfo is non-nil.
	GetAgentInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken, noErrorOnUnknownError bool) (*api.AgentInfo, error, bool /* return the error? */)
	// PollImmediateUntil should be used by the top-level polling, so that it can be gracefully interrupted
	// by the server when necessary.
	PollImmediateUntil(ctx context.Context, interval, maxConnectionAge time.Duration, condition ConditionFunc) error
}

type Factory interface {
	// New creates a new instance of a Module.
	New(*Config) (Module, error)
	// Name returns module's name.
	Name() string
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

func RoutingMetadata(agentId int64) metadata.MD {
	return metadata.MD{
		RoutingAgentIdMetadataKey: []string{strconv.FormatInt(agentId, 10)},
	}
}
