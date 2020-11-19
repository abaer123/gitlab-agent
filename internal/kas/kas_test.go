package kas

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_errtracker"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
)

const (
	token            = "abfaasdfasdfasdf"
	projectId        = "some/project"
	revision         = "507ebc6de9bcac25628aa7afd52802a91a0685d8"
	manifestRevision = "7afd52802a91a0685d8507ebc6de9bcac25628aa"

	maxConfigurationFileSize       = 128 * 1024
	maxGitopsManifestFileSize      = 128 * 1024
	maxGitopsTotalManifestFileSize = 1024 * 1024
	maxGitopsNumberOfPaths         = 10
	maxGitopsNumberOfFiles         = 200
)

func incomingCtx(ctx context.Context, t *testing.T) context.Context {
	creds := apiutil.NewTokenCredentials(token, false)
	meta, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	ctx = metadata.NewIncomingContext(ctx, metadata.New(meta))
	agentMeta, err := apiutil.AgentMetaFromRawContext(ctx)
	require.NoError(t, err)
	return apiutil.InjectAgentMeta(ctx, agentMeta)
}

func setupKas(t *testing.T) (*Server, *api.AgentInfo, *gomock.Controller, *mock_internalgitaly.MockPoolInterface, *mock_gitlab.MockClientInterface, *mock_errtracker.MockTracker) { // nolint: unparam
	k, mockCtrl, gitalyPool, gitlabClient, errTracker := setupKasBare(t)
	agentInfo := agentInfoObj()
	gitlabClient.EXPECT().
		GetAgentInfo(gomock.Any(), &agentInfo.Meta).
		Return(agentInfo, nil)

	return k, agentInfo, mockCtrl, gitalyPool, gitlabClient, errTracker
}

func setupKasBare(t *testing.T) (*Server, *gomock.Controller, *mock_internalgitaly.MockPoolInterface, *mock_gitlab.MockClientInterface, *mock_errtracker.MockTracker) {
	mockCtrl := gomock.NewController(t)
	gitalyPool := mock_internalgitaly.NewMockPoolInterface(mockCtrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(mockCtrl)
	errTracker := mock_errtracker.NewMockTracker(mockCtrl)

	k, cleanup, err := NewServer(Config{
		Log:                            zaptest.NewLogger(t),
		GitalyPool:                     gitalyPool,
		GitLabClient:                   gitlabClient,
		Registerer:                     prometheus.NewPedanticRegistry(),
		ErrorTracker:                   errTracker,
		AgentConfigurationPollPeriod:   10 * time.Minute,
		GitopsPollPeriod:               10 * time.Minute,
		MaxConfigurationFileSize:       maxConfigurationFileSize,
		MaxGitopsManifestFileSize:      maxGitopsManifestFileSize,
		MaxGitopsTotalManifestFileSize: maxGitopsTotalManifestFileSize,
		MaxGitopsNumberOfPaths:         maxGitopsNumberOfPaths,
		MaxGitopsNumberOfFiles:         maxGitopsNumberOfFiles,
		ConnectionMaxAge:               10 * time.Minute,
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	return k, mockCtrl, gitalyPool, gitlabClient, errTracker
}

func agentInfoObj() *api.AgentInfo {
	return &api.AgentInfo{
		Meta: api.AgentMeta{
			Token: token,
		},
		Id:   123,
		Name: "agent1",
		GitalyInfo: api.GitalyInfo{
			Address: "127.0.0.1:123123",
			Token:   "abc",
			Features: map[string]string{
				"bla": "true",
			},
		},
		Repository: gitalypb.Repository{
			StorageName:        "StorageName",
			RelativePath:       "RelativePath",
			GitObjectDirectory: "GitObjectDirectory",
			GlRepository:       "GlRepository",
			GlProjectPath:      "GlProjectPath",
		},
	}
}
