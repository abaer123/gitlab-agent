package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
)

type server struct {
	rpc.UnimplementedReverseTunnelServer
	api           modserver.API
	tunnelHandler reverse_tunnel.TunnelHandler
}

func (s *server) Connect(server rpc.ReverseTunnel_ConnectServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)
	ageCtx := grpctool.MaxConnectionAgeContextFromStream(server)
	agentInfo, err := s.api.GetAgentInfo(ageCtx, log, agentToken)
	if err != nil {
		return err // no wrap
	}
	return s.tunnelHandler.HandleTunnel(ageCtx, agentInfo, server)
}
