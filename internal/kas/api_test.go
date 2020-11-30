package kas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_errtracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ modserver.API = &API{}
)

func TestGetAgentInfoFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agentMeta := &api.AgentMeta{
		Token: mock_gitlab.AgentkToken,
	}
	l := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	gitlabClient := mock_gitlab.NewMockClientInterface(ctrl)
	errTracker := mock_errtracker.NewMockTracker(ctrl)
	apiObj := NewAPI(APIConfig{
		GitLabClient:           gitlabClient,
		ErrorTracker:           errTracker,
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, agentMeta, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, agentMeta, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, agentInfoApiPath, nil, agentMeta, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}),
		errTracker.EXPECT().
			Capture(matcher.ErrorEq("GetAgentInfo(): error kind: 0; status: 500"), gomock.Any()).
			DoAndReturn(func(err error, opts ...errortracking.CaptureOption) {
				cancel() // exception captured, cancel the context to stop the test
			}),
	)
	info, err, retErr := apiObj.GetAgentInfo(ctx, l, agentMeta, false)
	require.True(t, retErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Nil(t, info)

	info, err, retErr = apiObj.GetAgentInfo(ctx, l, agentMeta, false)
	require.True(t, retErr)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Nil(t, info)

	info, err, retErr = apiObj.GetAgentInfo(ctx, l, agentMeta, false)
	require.True(t, retErr)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Nil(t, info)
}

func TestGetAgentInfo(t *testing.T) {
	ctx, correlationId := mock_gitlab.CtxWithCorrelation(t)
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
		if !mock_gitlab.AssertGetRequestIsCorrect(t, w, r, correlationId) {
			return
		}

		mock_gitlab.RespondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	l := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	agentMeta := &api.AgentMeta{
		Token: mock_gitlab.AgentkToken,
	}
	apiObj := NewAPI(APIConfig{
		GitLabClient:           gitlab.NewClient(u, []byte(mock_gitlab.AuthSecretKey), mock_gitlab.ClientOptionsForTest()...),
		ErrorTracker:           mock_errtracker.NewMockTracker(ctrl),
		AgentInfoCacheTtl:      0, // no cache!
		AgentInfoCacheErrorTtl: 0,
	})
	agentInfo, err, retErr := apiObj.GetAgentInfo(ctx, l, agentMeta, false)
	require.False(t, retErr)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, agentInfo.ProjectId)
	assert.Equal(t, response.AgentId, agentInfo.Id)
	assert.Equal(t, response.AgentName, agentInfo.Name)

	mock_gitlab.AssertGitalyInfo(t, response.GitalyInfo, agentInfo.GitalyInfo)
	mock_gitlab.AssertGitalyRepository(t, response.GitalyRepository, agentInfo.Repository)
}
