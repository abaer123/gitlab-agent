package kasapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_errtracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ modserver.API = &serverAPI{}
)

func TestGetAgentInfoFailures_Forbidden(t *testing.T) {
	log, gitlabClient, errTracker, apiObj := setupApi(t)
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
		errTracker.EXPECT().
			Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 1; status: 403"), gomock.Any()),
	)
	info, err, retErr := apiObj.GetAgentInfo(context.Background(), log, testhelpers.AgentkToken, false)
	require.True(t, retErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfoFailures_Unauthorized(t *testing.T) {
	log, gitlabClient, errTracker, apiObj := setupApi(t)
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
		errTracker.EXPECT().
			Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 2; status: 401"), gomock.Any()),
	)
	info, err, retErr := apiObj.GetAgentInfo(context.Background(), log, testhelpers.AgentkToken, false)
	require.True(t, retErr)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfoFailures_InternalServerError(t *testing.T) {
	log, gitlabClient, errTracker, apiObj := setupApi(t)
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}),
		errTracker.EXPECT().
			Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 0; status: 500"), gomock.Any()),
	)
	info, err, retErr := apiObj.GetAgentInfo(context.Background(), log, testhelpers.AgentkToken, false)
	require.True(t, retErr)
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
	r := http.NewServeMux()
	r.HandleFunc(agentInfoApiPath, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		if !testhelpers.AssertGetRequestIsCorrect(t, w, r, correlationId) {
			return
		}

		testhelpers.RespondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	l := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	apiObj := newAPI(apiConfig{
		GitLabClient:           gitlab.NewClient(u, []byte(testhelpers.AuthSecretKey), mock_gitlab.ClientOptionsForTest()...),
		ErrorTracker:           mock_errtracker.NewMockTracker(ctrl),
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	agentInfo, err, retErr := apiObj.GetAgentInfo(ctx, l, testhelpers.AgentkToken, false)
	require.False(t, retErr)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, agentInfo.ProjectId)
	assert.Equal(t, response.AgentId, agentInfo.Id)
	assert.Equal(t, response.AgentName, agentInfo.Name)

	mock_gitlab.AssertGitalyInfo(t, response.GitalyInfo, agentInfo.GitalyInfo)
	mock_gitlab.AssertGitalyRepository(t, response.GitalyRepository, agentInfo.Repository)
}

func setupApi(t *testing.T) (*zap.Logger, *mock_gitlab.MockClientInterface, *mock_errtracker.MockTracker, *serverAPI) {
	log := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	gitlabClient := mock_gitlab.NewMockClientInterface(ctrl)
	errTracker := mock_errtracker.NewMockTracker(ctrl)
	apiObj := newAPI(apiConfig{
		GitLabClient:           gitlabClient,
		ErrorTracker:           errTracker,
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	return log, gitlabClient, errTracker, apiObj
}
