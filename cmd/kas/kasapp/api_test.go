package kasapp

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_errtracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ modserver.API = &serverAPI{}
)

func TestGetAgentInfoFailures_Forbidden(t *testing.T) {
	ctx, log, errTracker, apiObj := setupApi(t, http.StatusForbidden)
	errTracker.EXPECT().
		Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 1; status: 403"), gomock.Any())
	info, err := apiObj.GetAgentInfo(ctx, log, testhelpers.AgentkToken)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfoFailures_Unauthorized(t *testing.T) {
	ctx, log, errTracker, apiObj := setupApi(t, http.StatusUnauthorized)
	errTracker.EXPECT().
		Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 2; status: 401"), gomock.Any())
	info, err := apiObj.GetAgentInfo(ctx, log, testhelpers.AgentkToken)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfoFailures_InternalServerError(t *testing.T) {
	ctx, log, errTracker, apiObj := setupApi(t, http.StatusInternalServerError)
	errTracker.EXPECT().
		Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 0; status: 500"), gomock.Any())
	info, err := apiObj.GetAgentInfo(ctx, log, testhelpers.AgentkToken)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfo(t *testing.T) {
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	response := getAgentInfoResponse{
		ProjectId: 234,
		AgentId:   555,
		AgentName: "agent-x",
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
	l := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	gitLabClient := mock_gitlab.SetupClient(t, agentInfoApiPath, func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequestIsCorrect(t, r, correlationId)
		testhelpers.RespondWithJSON(t, w, response)
	})
	apiObj := newAPI(apiConfig{
		GitLabClient:           gitLabClient,
		ErrorTracker:           mock_errtracker.NewMockTracker(ctrl),
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	agentInfo, err := apiObj.GetAgentInfo(ctx, l, testhelpers.AgentkToken)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, agentInfo.ProjectId)
	assert.Equal(t, response.AgentId, agentInfo.Id)
	assert.Equal(t, response.AgentName, agentInfo.Name)

	mock_gitlab.AssertGitalyInfo(t, response.GitalyInfo, agentInfo.GitalyInfo)
	mock_gitlab.AssertGitalyRepository(t, response.GitalyRepository, agentInfo.Repository)
}

func setupApi(t *testing.T, statusCode int) (context.Context, *zap.Logger, *mock_errtracker.MockTracker, *serverAPI) {
	log := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	errTracker := mock_errtracker.NewMockTracker(ctrl)
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	gitLabClient := mock_gitlab.SetupClient(t, agentInfoApiPath, func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequestIsCorrect(t, r, correlationId)
		w.WriteHeader(statusCode)
	})
	apiObj := newAPI(apiConfig{
		GitLabClient:           gitLabClient,
		ErrorTracker:           errTracker,
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	return ctx, log, errTracker, apiObj
}
