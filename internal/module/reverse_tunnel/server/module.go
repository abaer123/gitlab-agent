package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/mathz"
)

const (
	maxConnectionAgeJitterPercent = 5
)

type module struct {
	rpc.UnimplementedReverseTunnelServer
	api              modserver.API
	maxConnectionAge time.Duration
	tunnelHandler    reverse_tunnel.TunnelHandler
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return reverse_tunnel.ModuleName
}

func (m *module) Connect(server rpc.ReverseTunnel_ConnectServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)
	ageCtx, cancel := context.WithTimeout(ctx, mathz.DurationWithJitter(m.maxConnectionAge, maxConnectionAgeJitterPercent))
	defer cancel()
	agentInfo, err, retErr := m.api.GetAgentInfo(ageCtx, log, agentToken, false)
	if retErr {
		return err // no wrap
	}
	return m.tunnelHandler.HandleTunnel(ageCtx, agentInfo, server)
}
