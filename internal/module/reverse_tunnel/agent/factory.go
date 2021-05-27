package agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"google.golang.org/grpc"
)

const (
	defaultNumConnections = 10

	connectionInitBackoff   = 10 * time.Second
	connectionMaxBackoff    = 5 * time.Minute
	connectionResetDuration = 10 * time.Minute
	connectionBackoffFactor = 2.0
	connectionJitter        = 1.0
)

type Factory struct {
	InternalServerConn grpc.ClientConnInterface
	NumConnections     int
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	sv, err := grpctool.NewStreamVisitor(&rpc.ConnectResponse{})
	if err != nil {
		return nil, err
	}
	numConnections := f.NumConnections
	if numConnections == 0 {
		numConnections = defaultNumConnections
	}
	featureChan := make(chan bool)
	config.Api.SubscribeToFeatureStatus(modagent.Tunnel, func(enabled bool) {
		featureChan <- enabled
	})
	client := rpc.NewReverseTunnelClient(config.KasConn)
	return &module{
		server:         config.Server,
		numConnections: numConnections,
		featureChan:    featureChan,
		connectionFactory: func(descriptor *info.AgentDescriptor) connectionInterface {
			return &connection{
				log:                config.Log,
				descriptor:         descriptor,
				client:             client,
				internalServerConn: f.InternalServerConn,
				streamVisitor:      sv,
				backoff: retry.NewExponentialBackoffFactory(
					connectionInitBackoff,
					connectionMaxBackoff,
					connectionResetDuration,
					connectionBackoffFactor,
					connectionJitter,
				),
			}
		},
	}, nil
}

func (f *Factory) Name() string {
	return reverse_tunnel.ModuleName
}
