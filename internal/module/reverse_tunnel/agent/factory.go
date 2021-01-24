package agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc"
)

const (
	defaultNumConnections = 10
	connectRetryPeriod    = 10 * time.Second
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
	return &module{
		log:                config.Log,
		server:             config.Server,
		client:             rpc.NewReverseTunnelClient(config.KasConn),
		internalServerConn: f.InternalServerConn,
		streamVisitor:      sv,
		connectRetryPeriod: connectRetryPeriod,
		numConnections:     numConnections,
	}, nil
}

func (f *Factory) Name() string {
	return reverse_tunnel.ModuleName
}
