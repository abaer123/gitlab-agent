package api

import (
	"context"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
)

const (
	ProjectInfoApiPath  = "/api/v4/internal/kubernetes/project_info"
	ProjectIdQueryParam = "id"
)

type ProjectInfoResponse struct {
	ProjectId        int64                   `json:"project_id"`
	GitalyInfo       gitlab.GitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitlab.GitalyRepository `json:"gitaly_repository"`
}

func GetProjectInfo(ctx context.Context, client gitlab.ClientInterface, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	response := ProjectInfoResponse{}
	err := client.Do(ctx,
		gitlab.WithPath(ProjectInfoApiPath),
		gitlab.WithQuery(url.Values{
			ProjectIdQueryParam: []string{projectId},
		}),
		gitlab.WithAgentToken(agentToken),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(&response)),
		gitlab.WithJWT(true),
	)
	if err != nil {
		return nil, err
	}
	return &api.ProjectInfo{
		ProjectId:  response.ProjectId,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil

}
