package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/cache"
)

type projectInfoClient struct {
	GitLabClient             gitlab.ClientInterface
	ProjectInfoCacheTtl      time.Duration
	ProjectInfoCacheErrorTtl time.Duration
	ProjectInfoCache         *cache.Cache
}

func (c *projectInfoClient) GetProjectInfo(ctx context.Context, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	if c.ProjectInfoCacheTtl == 0 {
		return gapi.GetProjectInfo(ctx, c.GitLabClient, agentToken, projectId)
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
		item.projectInfo, item.err = gapi.GetProjectInfo(ctx, c.GitLabClient, agentToken, projectId)
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

type projectInfoCacheKey struct {
	agentToken api.AgentToken
	projectId  string
}

// projectInfoCacheItem holds cached information about a project.
type projectInfoCacheItem struct {
	projectInfo *api.ProjectInfo
	err         error
}
