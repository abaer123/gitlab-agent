package gitlab

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
)

type UsageData struct {
	GitopsSyncCount int64 `json:"gitops_sync_count"`
}

type ClientInterface interface {
	GetAgentInfo(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error)
	GetProjectInfo(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error)
	SendUsage(ctx context.Context, data *UsageData) error
}
