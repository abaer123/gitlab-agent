package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
)

const (
	maxConnectionAgeJitterPercent = 5
)

type server struct {
	rpc.UnimplementedReverseTunnelServer
	api             modserver.API
	maxPollDuration time.Duration
	tunnelHandler   reverse_tunnel.TunnelHandler
}

func (s *server) Connect(server rpc.ReverseTunnel_ConnectServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)
	pollCtx, cancel := context.WithTimeout(ctx, mathz.DurationWithJitter(s.maxPollDuration, maxConnectionAgeJitterPercent))
	defer cancel()
	agentInfo, err := s.api.GetAgentInfo(pollCtx, log, agentToken)
	if err != nil {
		return err // no wrap
	}
	return s.tunnelHandler.HandleTunnel(pollCtx, agentInfo, server)
}
