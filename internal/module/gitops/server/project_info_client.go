package server

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/cache"
)

type projectInfoClient struct {
	GitLabClient     gitlab.ClientInterface
	ProjectInfoCache *cache.CacheWithErr
}

func (c *projectInfoClient) GetProjectInfo(ctx context.Context, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	key := projectInfoCacheKey{agentToken: agentToken, projectId: projectId}
	projectInfo, err := c.ProjectInfoCache.GetItem(ctx, key, func() (interface{}, error) {
		return gapi.GetProjectInfo(ctx, c.GitLabClient, agentToken, projectId)
	})
	if err != nil {
		return nil, err
	}
	return projectInfo.(*api.ProjectInfo), nil
}

type projectInfoCacheKey struct {
	agentToken api.AgentToken
	projectId  string
}
