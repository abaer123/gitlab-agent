package server

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type module struct {
	log          *zap.Logger
	api          modserver.API
	agentQuerier agent_tracker.Querier
}

func (m *module) GetConnectedAgents(ctx context.Context, req *rpc.GetConnectedAgentsRequest) (*rpc.GetConnectedAgentsResponse, error) {
	switch v := req.Request.(type) {
	case *rpc.GetConnectedAgentsRequest_AgentId:
		connectedAgentInfos, err := m.agentQuerier.GetConnectionsByAgentId(ctx, v.AgentId)
		if err != nil {
			m.api.HandleProcessingError(ctx, m.log, "GetConnectionsByAgentId() failed", err)
			return nil, status.Error(codes.Unavailable, "GetConnectionsByAgentId() failed")
		}
		return &rpc.GetConnectedAgentsResponse{
			Agents: connectedAgentInfos,
		}, nil
	case *rpc.GetConnectedAgentsRequest_ProjectId:
		connectedAgentInfos, err := m.agentQuerier.GetConnectionsByProjectId(ctx, v.ProjectId)
		if err != nil {
			m.api.HandleProcessingError(ctx, m.log, "GetConnectionsByProjectId() failed", err)
			return nil, status.Error(codes.Unavailable, "GetConnectionsByProjectId() failed")
		}
		return &rpc.GetConnectedAgentsResponse{
			Agents: connectedAgentInfos,
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
