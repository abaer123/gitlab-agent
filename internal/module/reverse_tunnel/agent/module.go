package agent

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

type module struct {
	log                *zap.Logger
	server             *grpc.Server
	client             rpc.ReverseTunnelClient
	internalServerConn grpc.ClientConnInterface
	streamVisitor      *grpctool.StreamVisitor
	connectRetryPeriod time.Duration
	numConnections     int
}

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	descriptor := m.agentDescriptor()
	var wg wait.Group
	defer wg.Wait()
	for i := 0; i < m.numConnections; i++ {
		conn := connection{
			log:                m.log,
			descriptor:         descriptor,
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

func (m *module) agentDescriptor() *rpc.AgentDescriptor {
	info := m.server.GetServiceInfo()
	services := make([]*rpc.AgentService, 0, len(info))
	for svcName, svcInfo := range info {
		methods := make([]*rpc.ServiceMethod, 0, len(svcInfo.Methods))
		for _, mInfo := range svcInfo.Methods {
			methods = append(methods, &rpc.ServiceMethod{
				Name: mInfo.Name,
			})
		}
		services = append(services, &rpc.AgentService{
			Name:    svcName,
			Methods: methods,
		})
	}
	return &rpc.AgentDescriptor{
		Services: services,
	}
}
