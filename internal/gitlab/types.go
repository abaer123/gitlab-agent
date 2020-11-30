package gitlab

import (
	"context"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type ClientInterface interface {
	DoJSON(ctx context.Context, method, path string, query url.Values, agentMeta *api.AgentMeta, body, response interface{}) error
}

// Some shared types below.

type GitalyInfo struct {
	Address  string            `json:"address"`
	Token    string            `json:"token"`
	Features map[string]string `json:"features"`
}

func (g *GitalyInfo) ToGitalyInfo() api.GitalyInfo {
	return api.GitalyInfo{
		Address:  g.Address,
		Token:    g.Token,
		Features: g.Features,
	}
}

type GitalyRepository struct {
	StorageName   string `json:"storage_name"`
	RelativePath  string `json:"relative_path"`
	GlRepository  string `json:"gl_repository"`
	GlProjectPath string `json:"gl_project_path"`
}

func (r *GitalyRepository) ToProtoRepository() gitalypb.Repository {
	return gitalypb.Repository{
		StorageName:   r.StorageName,
		RelativePath:  r.RelativePath,
		GlRepository:  r.GlRepository,
		GlProjectPath: r.GlProjectPath,
	}
}
