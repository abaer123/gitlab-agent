package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	rpc.UnimplementedAgentConfigurationServer
	api                        modserver.API
	gitaly                     gitaly.PoolInterface
	agentRegisterer            agent_tracker.Registerer
	maxConfigurationFileSize   int64
	getConfigurationBackoff    retry.BackoffManagerFactory
	getConfigurationPollPeriod time.Duration
}

func (s *server) GetConfiguration(req *rpc.ConfigurationRequest, server rpc.AgentConfiguration_GetConfigurationServer) error {
	connectedAgentInfo := &agent_tracker.ConnectedAgentInfo{
		AgentMeta:    req.AgentMeta,
		ConnectedAt:  timestamppb.Now(),
		ConnectionId: mathz.Int63(),
	}
	defer s.maybeUnregisterAgent(connectedAgentInfo)
	ctx := server.Context()
	log := grpctool.LoggerFromContext(ctx)
	agentToken := api.AgentTokenFromContext(ctx)
	p := pollJob{
		ctx:                      ctx,
		api:                      s.api,
		gitaly:                   s.gitaly,
		newConfigCb:              s.onNewConfig(server),
		maxConfigurationFileSize: s.maxConfigurationFileSize,
		lastProcessedCommitId:    req.CommitId,
	}
	return s.api.PollWithBackoff(server, s.getConfigurationBackoff(), true, s.getConfigurationPollPeriod, func() (error, retry.AttemptResult) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		agentInfo, err := s.api.GetAgentInfo(ctx, log, agentToken)
		if err != nil {
			return err, retry.Done
		}
		s.maybeRegisterAgent(ctx, connectedAgentInfo, agentInfo)
		logWithFields := log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(agentInfo.Repository.GlProjectPath))
		return p.Attempt(logWithFields, agentInfo)
	})
}

func (s *server) onNewConfig(server rpc.AgentConfiguration_GetConfigurationServer) newConfigCb {
	return func(log *zap.Logger, config *agentcfg.AgentConfiguration, commitId string) (error, retry.AttemptResult) {
		err := server.Send(&rpc.ConfigurationResponse{
			Configuration: config,
			CommitId:      commitId,
		})
		if err != nil {
			return s.api.HandleSendError(log, "Config: failed to send config", err), retry.Done
		}
		return nil, retry.Continue
	}
}

func (s *server) maybeRegisterAgent(ctx context.Context, connectedAgentInfo *agent_tracker.ConnectedAgentInfo, agentInfo *api.AgentInfo) {
	if connectedAgentInfo.AgentId != 0 {
		return
	}
	connectedAgentInfo.AgentId = agentInfo.Id
	connectedAgentInfo.ProjectId = agentInfo.ProjectId
	s.agentRegisterer.RegisterConnection(ctx, connectedAgentInfo)
}

func (s *server) maybeUnregisterAgent(connectedAgentInfo *agent_tracker.ConnectedAgentInfo) {
	if connectedAgentInfo.AgentId == 0 {
		return
	}
	s.agentRegisterer.UnregisterConnection(context.Background(), connectedAgentInfo)
}
