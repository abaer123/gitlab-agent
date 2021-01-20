package agent

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ modagent.Module  = &module{}
	_ modagent.Factory = &Factory{}
)

type module struct {
	log                *zap.Logger
	client             rpc.ReverseTunnelClient
	internalServerConn grpc.ClientConnInterface
	streamVisitor      *grpctool.StreamVisitor
	connectRetryPeriod time.Duration
	numConnections     int
}

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	var wg wait.Group
	defer wg.Wait()
	for i := 0; i < m.numConnections; i++ {
		conn := connection{
			log:                m.log,
			client:             m.client,
			internalServerConn: m.internalServerConn,
			streamVisitor:      m.streamVisitor,
			connectRetryPeriod: m.connectRetryPeriod,
		}
		wg.StartWithContext(ctx, conn.Run)
	}
	return nil
}

func (m *module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	return nil
}

func (m *module) Name() string {
	return reverse_tunnel.ModuleName
}
