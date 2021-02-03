package server

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type module struct {
	api          modserver.API
	agentQuerier agent_tracker.Querier
}

func (m *module) GetConnectedAgents(ctx context.Context, req *rpc.GetConnectedAgentsRequest) (*rpc.GetConnectedAgentsResponse, error) {
	log := grpctool.LoggerFromContext(ctx)
	switch v := req.Request.(type) {
	case *rpc.GetConnectedAgentsRequest_AgentId:
		var infos agent_tracker.ConnectedAgentInfoCollector
		err := m.agentQuerier.GetConnectionsByAgentId(ctx, v.AgentId, infos.Collect)
		if err != nil {
			m.api.HandleProcessingError(ctx, log, "GetConnectionsByAgentId() failed", err)
			return nil, status.Error(codes.Unavailable, "GetConnectionsByAgentId() failed")
		}
		return &rpc.GetConnectedAgentsResponse{
			Agents: infos,
		}, nil
	case *rpc.GetConnectedAgentsRequest_ProjectId:
		var infos agent_tracker.ConnectedAgentInfoCollector
		err := m.agentQuerier.GetConnectionsByProjectId(ctx, v.ProjectId, infos.Collect)
		if err != nil {
			m.api.HandleProcessingError(ctx, log, "GetConnectionsByProjectId() failed", err)
			return nil, status.Error(codes.Unavailable, "GetConnectionsByProjectId() failed")
		}
		return &rpc.GetConnectedAgentsResponse{
			Agents: infos,
		}, nil
	default:
		// Should never happen
		return nil, status.Errorf(codes.InvalidArgument, "Unexpected field type: %T", req.Request)
	}
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return agent_tracker.ModuleName
}
