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
	api                     modserver.API
	maxConnectionAge        time.Duration
	tunnelConnectionHandler reverse_tunnel.TunnelConnectionHandler
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
	agentInfo, err, retErr := m.api.GetAgentInfo(ctx, log, agentToken, false)
	if retErr {
		return err // no wrap
	}
	ageCtx, cancel := context.WithTimeout(ctx, mathz.DurationWithJitter(m.maxConnectionAge, maxConnectionAgeJitterPercent))
	defer cancel()
	return m.tunnelConnectionHandler.HandleTunnelConnection(ageCtx, agentInfo, server)
}
