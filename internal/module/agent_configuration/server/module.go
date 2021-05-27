package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type module struct {
	rpc.UnimplementedAgentConfigurationServer
	api                        modserver.API
	gitaly                     gitaly.PoolInterface
	agentRegisterer            agent_tracker.Registerer
	maxConfigurationFileSize   int64
	getConfigurationBackoff    retry.BackoffManagerFactory
	getConfigurationPollPeriod time.Duration
	maxConnectionAge           time.Duration
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
	return m.api.PollWithBackoff(ctx, m.getConfigurationBackoff(), true, m.maxConnectionAge, m.getConfigurationPollPeriod, p.Attempt)
}
