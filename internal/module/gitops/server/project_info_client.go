package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
)

const (
	projectInfoApiPath  = "/api/v4/internal/kubernetes/project_info"
	projectIdQueryParam = "id"
)

type projectInfoClient struct {
	GitLabClient             gitlab.ClientInterface
	ProjectInfoCacheTtl      time.Duration
	ProjectInfoCacheErrorTtl time.Duration
	ProjectInfoCache         *cache.Cache
}

func (c *projectInfoClient) GetProjectInfo(ctx context.Context, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	if c.ProjectInfoCacheTtl == 0 {
		return c.getProjectInfoDirect(ctx, agentToken, projectId)
	}
	c.ProjectInfoCache.EvictExpiredEntries()
	entry := c.ProjectInfoCache.GetOrCreateCacheEntry(projectInfoCacheKey{
		agentToken: agentToken,
		projectId:  projectId,
	})
	if !entry.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.Unlock()
	var item projectInfoCacheItem
	if entry.IsNeedRefreshLocked() {
		item.projectInfo, item.err = c.getProjectInfoDirect(ctx, agentToken, projectId)
		var ttl time.Duration
		if item.err == nil {
			ttl = c.ProjectInfoCacheTtl
		} else {
			ttl = c.ProjectInfoCacheErrorTtl
		}
		entry.Item = item
		entry.Expires = time.Now().Add(ttl)
	} else {
		item = entry.Item.(projectInfoCacheItem)
	}
	return item.projectInfo, item.err
}

func (c *projectInfoClient) getProjectInfoDirect(ctx context.Context, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	query := url.Values{
		projectIdQueryParam: []string{projectId},
	}
	response := projectInfoResponse{}
	err := c.GitLabClient.DoJSON(ctx, http.MethodGet, projectInfoApiPath, query, agentToken, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.ProjectInfo{
		ProjectId:  response.ProjectId,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

type projectInfoResponse struct {
	ProjectId        int64                   `json:"project_id"`
	GitalyInfo       gitlab.GitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitlab.GitalyRepository `json:"gitaly_repository"`
}

type projectInfoCacheKey struct {
	agentToken api.AgentToken
	projectId  string
}

// projectInfoCacheItem holds cached information about a project.
type projectInfoCacheItem struct {
	projectInfo *api.ProjectInfo
	err         error
}
