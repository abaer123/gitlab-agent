package kas

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/sentryapi/mock_sentryapi"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	token     = "abfaasdfasdfasdf"
	projectId = "some/project"
	revision  = "507ebc6de9bcac25628aa7afd52802a91a0685d8"

	infoRefsData = `001e# service=git-upload-pack
00000148` + revision + ` HEAD` + "\x00" + `multi_ack thin-pack side-band side-band-64k ofs-delta shallow deepen-since deepen-not deepen-relative no-progress include-tag multi_ack_detailed allow-tip-sha1-in-want allow-reachable-sha1-in-want no-done symref=HEAD:refs/heads/master filter object-format=sha1 agent=git/2.28.0
003f` + revision + ` refs/heads/master
0044` + revision + ` refs/heads/test-branch
0000`
)

func TestYAMLToConfigurationAndBack(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		given, expected string
	}{
		{
			given: `{}
`, // empty config
			expected: `{}
`,
		},
		{
			given: `gitops: {}
`,
			expected: `gitops: {}
`,
		},
		{
			given: `gitops:
  manifest_projects: []
`,
			expected: `gitops: {}
`, // empty slice is omitted
		},
		{
			expected: `gitops:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
			given: `gitops:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			config, err := parseYAMLToConfiguration([]byte(tc.given)) // nolint: scopelint
			require.NoError(t, err)
			configJson, err := protojson.Marshal(config)
			require.NoError(t, err)
			configYaml, err := yaml.JSONToYAML(configJson)
			require.NoError(t, err)
			diff := cmp.Diff(tc.expected, string(configYaml)) // nolint: scopelint
			assert.Empty(t, diff)
		})
	}
}

func TestGetConfiguration(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, _, _ := setupKas(t)
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: &agentInfo.Repository,
		Revision:   []byte(revision),
		Path:       []byte(agentConfigurationDirectory + "/" + agentInfo.Name + "/" + agentConfigurationFileName),
		Limit:      fileSizeLimit,
	}
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &agentInfo.Repository,
	}
	configFile := sampleConfig()
	resp := mock_agentrpc.NewMockKas_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	resp.EXPECT().
		Send(matcher.ProtoEq(t, &agentrpc.ConfigurationResponse{
			Configuration: &agentcfg.AgentConfiguration{
				Gitops: configFile.Gitops,
			},
			CommitId: revision,
		})).
		DoAndReturn(func(resp *agentrpc.ConfigurationResponse) error {
			cancel() // stop streaming call after the first response has been sent
			return nil
		})
	httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		SmartHTTPServiceClient(gomock.Any(), &agentInfo.GitalyInfo).
		Return(httpClient, nil)
	mockInfoRefsUploadPack(t, mockCtrl, httpClient, infoRefsReq, []byte(infoRefsData))
	commitClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		CommitServiceClient(gomock.Any(), &agentInfo.GitalyInfo).
		Return(commitClient, nil)
	mockTreeEntry(t, mockCtrl, commitClient, treeEntryReq, configToBytes(t, configFile))
	err := a.GetConfiguration(&agentrpc.ConfigurationRequest{}, resp)
	require.NoError(t, err)
}

func TestGetConfigurationResumeConnection(t *testing.T) {
	t.Parallel()
	// we check that nothing gets sent back when the request with the last commit id comes
	// so we just wait to see that nothing happens
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, _, _ := setupKas(t)
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &agentInfo.Repository,
	}
	resp := mock_agentrpc.NewMockKas_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		SmartHTTPServiceClient(gomock.Any(), &agentInfo.GitalyInfo).
		Return(httpClient, nil)
	mockInfoRefsUploadPack(t, mockCtrl, httpClient, infoRefsReq, []byte(infoRefsData))
	err := a.GetConfiguration(&agentrpc.ConfigurationRequest{
		CommitId: revision, // same commit id
	}, resp)
	require.NoError(t, err)
}

func TestGetConfigurationGitLabClientFailures(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	agentMeta := api.AgentMeta{
		Token: token,
	}
	k, mockCtrl, _, gitlabClient, _ := setupKasBare(t)
	gomock.InOrder(
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), &agentMeta).
			Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), &agentMeta).
			Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), &agentMeta).
			DoAndReturn(func(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
				cancel()
				return nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}
			}),
	)
	resp := mock_agentrpc.NewMockKas_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	err := k.GetConfiguration(&agentrpc.ConfigurationRequest{}, resp)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	err = k.GetConfiguration(&agentrpc.ConfigurationRequest{}, resp)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	err = k.GetConfiguration(&agentrpc.ConfigurationRequest{}, resp)
	require.NoError(t, err)
}

func TestGetObjectsToSynchronizeGitLabClientFailures(t *testing.T) {
	t.Parallel()
	t.Run("GetAgentInfo failures", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		k, mockCtrl, _, gitlabClient, _ := setupKasBare(t)
		agentInfo := agentInfoObj()

		gomock.InOrder(
			gitlabClient.EXPECT().
				GetAgentInfo(gomock.Any(), &agentInfo.Meta).
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
			gitlabClient.EXPECT().
				GetAgentInfo(gomock.Any(), &agentInfo.Meta).
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
			gitlabClient.EXPECT().
				GetAgentInfo(gomock.Any(), &agentInfo.Meta).
				DoAndReturn(func(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
					cancel()
					return nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}
				}),
		)

		resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
		resp.EXPECT().
			Context().
			Return(incomingCtx(ctx, t)).
			MinTimes(1)
		err := k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		err = k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		err = k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
	t.Run("GetProjectInfo failures", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		k, mockCtrl, _, gitlabClient, _ := setupKasBare(t)
		agentInfo := agentInfoObj()
		gitlabClient.EXPECT().
			GetAgentInfo(gomock.Any(), &agentInfo.Meta).
			Return(agentInfo, nil).
			Times(3)

		gomock.InOrder(
			gitlabClient.EXPECT().
				GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
			gitlabClient.EXPECT().
				GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
			gitlabClient.EXPECT().
				GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
				DoAndReturn(func(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
					cancel()
					return nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}
				}),
		)
		resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
		resp.EXPECT().
			Context().
			Return(incomingCtx(ctx, t)).
			MinTimes(1)
		err := k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		err = k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		err = k.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
		require.NoError(t, err)
	})
}

func TestGetObjectsToSynchronize(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, gitlabClient, _ := setupKas(t)
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 1})).
		Return(nil)
	a.usageReportingPeriod = 10 * time.Millisecond

	objects := []runtime.Object{
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "map1",
			},
			Data: map[string]string{
				"key": "value",
			},
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
			},
		},
	}
	objectsYAML := kube_testing.ObjsToYAML(t, objects...)
	projectInfo := projectInfo()
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: &projectInfo.Repository,
		Revision:   []byte(revision),
		Path:       []byte("manifest.yaml"),
		Limit:      fileSizeLimit,
	}
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &projectInfo.Repository,
	}
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	resp.EXPECT().
		Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
			CommitId: revision,
			Objects: []*agentrpc.ObjectToSynchronize{
				{
					Object: objectsYAML,
					Source: "manifest.yaml",
				},
			},
		})).
		DoAndReturn(func(resp *agentrpc.ObjectsToSynchronizeResponse) error {
			cancel() // stop streaming call after the first response has been sent
			return nil
		})
	gitlabClient.EXPECT().
		GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
		Return(projectInfo, nil)
	httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		SmartHTTPServiceClient(gomock.Any(), &projectInfo.GitalyInfo).
		Return(httpClient, nil)
	mockInfoRefsUploadPack(t, mockCtrl, httpClient, infoRefsReq, []byte(infoRefsData))
	commitClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		CommitServiceClient(gomock.Any(), &projectInfo.GitalyInfo).
		Return(commitClient, nil)
	mockTreeEntry(t, mockCtrl, commitClient, treeEntryReq, objectsYAML)
	err := a.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
	require.NoError(t, err)

	ctxRun, cancelRun := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelRun()
	a.Run(ctxRun)
}

func TestGetObjectsToSynchronizeResumeConnection(t *testing.T) {
	t.Parallel()
	// we check that nothing gets sent back when the request with the last commit id comes
	// so we just wait to see that nothing happens
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, gitlabClient, _ := setupKas(t)
	projectInfo := projectInfo()
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &projectInfo.Repository,
	}
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	gitlabClient.EXPECT().
		GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
		Return(projectInfo, nil)
	httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		SmartHTTPServiceClient(gomock.Any(), &projectInfo.GitalyInfo).
		Return(httpClient, nil)
	mockInfoRefsUploadPack(t, mockCtrl, httpClient, infoRefsReq, []byte(infoRefsData))
	commitClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	gitalyPool.EXPECT().
		CommitServiceClient(gomock.Any(), &projectInfo.GitalyInfo).
		Return(commitClient, nil)
	err := a.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		CommitId:  revision,
	}, resp)
	require.NoError(t, err)
}

func TestSendUsage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, _, _, gitlabClient, _ := setupKasBare(t)
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
		Return(nil)
	k.usageMetrics.gitopsSyncCount = 5

	// Send accumulated counters
	require.NoError(t, k.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, k.sendUsageInternal(ctx))
}

func TestSendUsageFailure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedErr := errors.New("expected error")
	k, _, _, gitlabClient, sentryHub := setupKasBare(t)
	sentryHub.EXPECT().
		CaptureException(expectedErr).
		DoAndReturn(func(err error) *sentry.EventID {
			cancel() // exception captured, cancel the context to stop the test
			return nil
		})
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
		Return(expectedErr)
	k.usageMetrics.gitopsSyncCount = 5

	k.sendUsage(ctx)
}

func TestSendUsageRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, _, _, gitlabClient, _ := setupKasBare(t)
	gomock.InOrder(
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
			Return(errors.New("expected error")),
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 6})).
			Return(nil),
	)
	k.usageMetrics.gitopsSyncCount = 5

	// Try to send accumulated counters, fail
	require.EqualError(t, k.sendUsageInternal(ctx), "expected error")

	k.usageMetrics.gitopsSyncCount++

	// Try again and succeed
	require.NoError(t, k.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, k.sendUsageInternal(ctx))
}

func mockTreeEntry(t *testing.T, mockCtrl *gomock.Controller, commitClient *mock_gitaly.MockCommitServiceClient, req *gitalypb.TreeEntryRequest, data []byte) {
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(mockCtrl)
	// Emulate streaming response
	resp1 := &gitalypb.TreeEntryResponse{
		Data: data[:1],
	}
	resp2 := &gitalypb.TreeEntryResponse{
		Data: data[1:],
	}
	gomock.InOrder(
		treeEntryClient.EXPECT().
			Recv().
			Return(resp1, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(resp2, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	commitClient.EXPECT().
		TreeEntry(gomock.Any(), matcher.ProtoEq(t, req)).
		Return(treeEntryClient, nil)
}

func mockInfoRefsUploadPack(t *testing.T, mockCtrl *gomock.Controller, httpClient *mock_gitaly.MockSmartHTTPServiceClient, infoRefsReq *gitalypb.InfoRefsRequest, data []byte) {
	infoRefsClient := mock_gitaly.NewMockSmartHTTPService_InfoRefsUploadPackClient(mockCtrl)
	// Emulate streaming response
	resp1 := &gitalypb.InfoRefsResponse{
		Data: data[:1],
	}
	resp2 := &gitalypb.InfoRefsResponse{
		Data: data[1:],
	}
	gomock.InOrder(
		infoRefsClient.EXPECT().
			Recv().
			Return(resp1, nil),
		infoRefsClient.EXPECT().
			Recv().
			Return(resp2, nil),
		infoRefsClient.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	httpClient.EXPECT().
		InfoRefsUploadPack(gomock.Any(), matcher.ProtoEq(t, infoRefsReq)).
		Return(infoRefsClient, nil)
}

func incomingCtx(ctx context.Context, t *testing.T) context.Context {
	creds := apiutil.NewTokenCredentials(token, false)
	meta, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	ctx = metadata.NewIncomingContext(ctx, metadata.New(meta))
	agentMeta, err := apiutil.AgentMetaFromRawContext(ctx)
	require.NoError(t, err)
	return apiutil.InjectAgentMeta(ctx, agentMeta)
}

func configToBytes(t *testing.T, configFile *agentcfg.ConfigurationFile) []byte {
	configJson, err := protojson.Marshal(configFile)
	require.NoError(t, err)
	configYaml, err := yaml.JSONToYAML(configJson)
	require.NoError(t, err)
	return configYaml
}

func sampleConfig() *agentcfg.ConfigurationFile {
	return &agentcfg.ConfigurationFile{
		Gitops: &agentcfg.GitopsCF{
			ManifestProjects: []*agentcfg.ManifestProjectCF{
				{
					Id: projectId,
				},
			},
		},
	}
}

func projectInfo() *api.ProjectInfo {
	return &api.ProjectInfo{
		ProjectId: 234,
		GitalyInfo: api.GitalyInfo{
			Address: "127.0.0.1:321321",
			Token:   "cba",
			Features: map[string]string{
				"bla": "false",
			},
		},
		Repository: gitalypb.Repository{
			StorageName:        "StorageName1",
			RelativePath:       "RelativePath1",
			GitObjectDirectory: "GitObjectDirectory1",
			GlRepository:       "GlRepository1",
			GlProjectPath:      "GlProjectPath1",
		},
	}
}

func setupKas(t *testing.T) (*Server, *api.AgentInfo, *gomock.Controller, *mock_gitaly.MockPoolInterface, *mock_gitlab.MockClientInterface, *mock_sentryapi.MockHub) { // nolint: unparam
	k, mockCtrl, gitalyPool, gitlabClient, sentryHub := setupKasBare(t)
	agentInfo := agentInfoObj()
	gitlabClient.EXPECT().
		GetAgentInfo(gomock.Any(), &agentInfo.Meta).
		Return(agentInfo, nil)

	return k, agentInfo, mockCtrl, gitalyPool, gitlabClient, sentryHub
}

func setupKasBare(t *testing.T) (*Server, *gomock.Controller, *mock_gitaly.MockPoolInterface, *mock_gitlab.MockClientInterface, *mock_sentryapi.MockHub) {
	mockCtrl := gomock.NewController(t)
	gitalyPool := mock_gitaly.NewMockPoolInterface(mockCtrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(mockCtrl)
	sentryHub := mock_sentryapi.NewMockHub(mockCtrl)

	k, cleanup, err := NewServer(Config{
		Log:                          zaptest.NewLogger(t),
		GitalyPool:                   gitalyPool,
		GitLabClient:                 gitlabClient,
		AgentConfigurationPollPeriod: 10 * time.Minute,
		GitopsPollPeriod:             10 * time.Minute,
		Registerer:                   prometheus.NewPedanticRegistry(),
		Sentry:                       sentryHub,
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	return k, mockCtrl, gitalyPool, gitlabClient, sentryHub
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
