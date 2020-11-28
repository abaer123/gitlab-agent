package kas

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_errtracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetAgentInfoFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agentMeta := &api.AgentMeta{
		Token: token,
	}
	l := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	gitlabClient := mock_gitlab.NewMockClientInterface(ctrl)
	errTracker := mock_errtracker.NewMockTracker(ctrl)
	apiObj := &API{
		GitLabClient: gitlabClient,
		ErrorTracker: errTracker,
	}
	gomock.InOrder(
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), agentMeta).
			Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), agentMeta).
			Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), agentMeta).
			Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}),
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
