package modserver

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
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
	GetAgentInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken) (*api.AgentInfo, error)
	// PollWithBackoff runs f every duration given by BackoffManager.
	//
	// PollWithBackoff should be used by the top-level polling, so that it can be gracefully interrupted
	// by the server when necessary. E.g. when stream is nearing it's max connection age or program needs to
	// be shut down.
	// If sliding is true, the period is computed after f runs. If it is false then
	// period includes the runtime for f.
	// It returns when:
	// - stream's context is cancelled or max connection age has been reached. nil is returned in this case.
	// - f returns Done. error from f is returned in this case.
	PollWithBackoff(stream grpc.ServerStream, cfg retry.PollConfig, f retry.PollWithBackoffFunc) error
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

func RoutingMetadata(agentId int64) metadata.MD {
	return metadata.MD{
		RoutingAgentIdMetadataKey: []string{strconv.FormatInt(agentId, 10)},
	}
}
