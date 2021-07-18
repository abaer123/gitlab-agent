package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
)

type Factory struct {
	TunnelHandler reverse_tunnel.TunnelHandler
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	rpc.RegisterReverseTunnelServer(config.AgentServer, &server{
		api:           config.Api,
		tunnelHandler: f.TunnelHandler,
	})
	return &module{}, nil
}

func (f *Factory) Name() string {
	return reverse_tunnel.ModuleName
}
