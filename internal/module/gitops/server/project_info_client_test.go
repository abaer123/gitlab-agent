package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
)

func TestGetProjectInfo(t *testing.T) {
	const (
		projectId = "bla/bla"
	)
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	response := projectInfoResponse{
		ProjectId: 234,
		GitalyInfo: gitlab.GitalyInfo{
			Address: "example.com",
			Token:   "123123",
			Features: map[string]string{
				"a": "b",
			},
		},
		GitalyRepository: gitlab.GitalyRepository{
			StorageName:   "234",
			RelativePath:  "123",
			GlRepository:  "254634",
			GlProjectPath: "64662",
		},
	}
	gitLabClient := mock_gitlab.SetupClient(t, projectInfoApiPath, func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertRequestMethod(t, r, http.MethodGet)
		testhelpers.AssertGetJsonRequestIsCorrect(t, r, correlationId)
		assert.Equal(t, projectId, r.URL.Query().Get(projectIdQueryParam))

		testhelpers.RespondWithJSON(t, w, response)
	})
	pic := projectInfoClient{
		GitLabClient:             gitLabClient,
		ProjectInfoCacheTtl:      0, // no cache
		ProjectInfoCacheErrorTtl: 0,
		ProjectInfoCache:         cache.New(0),
	}

	projInfo, err := pic.GetProjectInfo(ctx, testhelpers.AgentkToken, projectId)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, projInfo.ProjectId)
	mock_gitlab.AssertGitalyInfo(t, response.GitalyInfo, projInfo.GitalyInfo)
	mock_gitlab.AssertGitalyRepository(t, response.GitalyRepository, projInfo.Repository)
}
