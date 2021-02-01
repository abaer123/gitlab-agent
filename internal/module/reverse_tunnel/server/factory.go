package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
)

type Factory struct {
	TunnelConnectionHandler reverse_tunnel.TunnelConnectionHandler
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	m := &module{
		api:                     config.Api,
		maxConnectionAge:        config.Config.Agent.Listen.MaxConnectionAge.AsDuration(),
		tunnelConnectionHandler: f.TunnelConnectionHandler,
	}
	rpc.RegisterReverseTunnelServer(config.AgentServer, m)
	return m, nil
}

func (f *Factory) Name() string {
	return reverse_tunnel.ModuleName
}
