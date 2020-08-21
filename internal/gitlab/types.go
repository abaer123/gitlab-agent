package gitlab

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
)

type ClientInterface interface {
	GetAgentInfo(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error)
	GetProjectInfo(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error)
}
