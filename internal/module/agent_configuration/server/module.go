package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/mathz"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type module struct {
	api                          modserver.API
	gitaly                       gitaly.PoolInterface
	agentRegisterer              agent_tracker.Registerer
	maxConfigurationFileSize     int64
	agentConfigurationPollPeriod time.Duration
	maxConnectionAge             time.Duration
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return agent_configuration.ModuleName
}

func (m *module) GetConfiguration(req *rpc.ConfigurationRequest, server rpc.AgentConfiguration_GetConfigurationServer) error {
	ctx := server.Context()
	p := pollJob{
		ctx:                      ctx,
		log:                      grpctool.LoggerFromContext(ctx),
		api:                      m.api,
		gitaly:                   m.gitaly,
		agentRegisterer:          m.agentRegisterer,
		server:                   server,
		agentToken:               api.AgentTokenFromContext(ctx),
		maxConfigurationFileSize: m.maxConfigurationFileSize,
		lastProcessedCommitId:    req.CommitId,
		connectedAgentInfo: &agent_tracker.ConnectedAgentInfo{
			AgentMeta:    req.AgentMeta,
			ConnectedAt:  timestamppb.Now(),
			ConnectionId: mathz.Int63(),
		},
	}
	defer p.Cleanup()
	return m.api.PollImmediateUntil(ctx, m.agentConfigurationPollPeriod, m.maxConnectionAge, p.Attempt)
}
