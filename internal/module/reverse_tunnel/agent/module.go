package agent

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
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

func (m *module) agentDescriptor() *info.AgentDescriptor {
	serverInfo := m.server.GetServiceInfo()
	services := make([]*info.Service, 0, len(serverInfo))
	for svcName, svcInfo := range serverInfo {
		methods := make([]*info.Method, 0, len(svcInfo.Methods))
		for _, mInfo := range svcInfo.Methods {
			methods = append(methods, &info.Method{
				Name: mInfo.Name,
			})
		}
		services = append(services, &info.Service{
			Name:    svcName,
			Methods: methods,
		})
	}
	return &info.AgentDescriptor{
		Services: services,
	}
}
