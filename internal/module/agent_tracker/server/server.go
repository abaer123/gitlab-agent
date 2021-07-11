package server

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	rpc.UnimplementedAgentTrackerServer
	api          modserver.API
	agentQuerier agent_tracker.Querier
}

func (s *server) GetConnectedAgents(ctx context.Context, req *rpc.GetConnectedAgentsRequest) (*rpc.GetConnectedAgentsResponse, error) {
	log := grpctool.LoggerFromContext(ctx)
	switch v := req.Request.(type) {
	case *rpc.GetConnectedAgentsRequest_AgentId:
		var infos agent_tracker.ConnectedAgentInfoCollector
		err := s.agentQuerier.GetConnectionsByAgentId(ctx, v.AgentId, infos.Collect)
		if err != nil {
			s.api.HandleProcessingError(ctx, log, "GetConnectionsByAgentId() failed", err)
			return nil, status.Error(codes.Unavailable, "GetConnectionsByAgentId() failed")
		}
		return &rpc.GetConnectedAgentsResponse{
			Agents: infos,
		}, nil
	case *rpc.GetConnectedAgentsRequest_ProjectId:
		var infos agent_tracker.ConnectedAgentInfoCollector
		err := s.agentQuerier.GetConnectionsByProjectId(ctx, v.ProjectId, infos.Collect)
		if err != nil {
			s.api.HandleProcessingError(ctx, log, "GetConnectionsByProjectId() failed", err)
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
