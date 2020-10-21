package kas

import (
	"context"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly/mock_gitalypool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
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
	a, agentInfo, mockCtrl, gitalyPool, _ := setupKas(ctx, t)
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
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockKas_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
		MinTimes(1)
	resp.EXPECT().
		Send(matcher.ProtoEq(t, &agentrpc.ConfigurationResponse{
			Configuration: &agentcfg.AgentConfiguration{
				Gitops: configFile.Gitops,
			},
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
	a, agentInfo, mockCtrl, gitalyPool, _ := setupKas(ctx, t)
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &agentInfo.Repository,
	}
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockKas_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
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

func TestGetObjectsToSynchronize(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, gitlabClient := setupKas(ctx, t)
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
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
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
	a, agentInfo, mockCtrl, gitalyPool, gitlabClient := setupKas(ctx, t)
	projectInfo := projectInfo()
	infoRefsReq := &gitalypb.InfoRefsRequest{
		Repository: &projectInfo.Repository,
	}
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
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
	mockCtrl := gomock.NewController(t)
	gitalyPool := mock_gitalypool.NewMockPoolInterface(mockCtrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(mockCtrl)
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
		Return(nil)

	s, cleanup, err := NewServer(ServerConfig{
		Context:                      ctx,
		Log:                          zaptest.NewLogger(t),
		GitalyPool:                   gitalyPool,
		GitLabClient:                 gitlabClient,
		AgentConfigurationPollPeriod: 10 * time.Minute,
		GitopsPollPeriod:             10 * time.Minute,
		UsageReportingPeriod:         10 * time.Millisecond,
		Registerer:                   prometheus.NewPedanticRegistry(),
	})
	require.NoError(t, err)
	defer cleanup()
	s.usageMetrics.gitopsSyncCount = 5

	// Send accumulated counters
	require.NoError(t, s.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, s.sendUsageInternal(ctx))
}

func TestSendUsageRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	gitalyPool := mock_gitalypool.NewMockPoolInterface(mockCtrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(mockCtrl)
	gomock.InOrder(
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
			Return(errors.New("expected error")),
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 6})).
			Return(nil),
	)

	s, cleanup, err := NewServer(ServerConfig{
		Context:                      ctx,
		Log:                          zaptest.NewLogger(t),
		GitalyPool:                   gitalyPool,
		GitLabClient:                 gitlabClient,
		AgentConfigurationPollPeriod: 10 * time.Minute,
		GitopsPollPeriod:             10 * time.Minute,
		UsageReportingPeriod:         10 * time.Millisecond,
		Registerer:                   prometheus.NewPedanticRegistry(),
	})
	require.NoError(t, err)
	defer cleanup()
	s.usageMetrics.gitopsSyncCount = 5

	// Try to send accumulated counters, fail
	require.EqualError(t, s.sendUsageInternal(ctx), "expected error")

	s.usageMetrics.gitopsSyncCount++

	// Try again and succeed
	require.NoError(t, s.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, s.sendUsageInternal(ctx))
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

func setupKas(ctx context.Context, t *testing.T) (*Server, *api.AgentInfo, *gomock.Controller, *mock_gitalypool.MockPoolInterface, *mock_gitlab.MockClientInterface) {
	agentMeta := api.AgentMeta{
		Token: token,
	}
	agentInfo := &api.AgentInfo{
		Meta: agentMeta,
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

	mockCtrl := gomock.NewController(t)
	gitalyPool := mock_gitalypool.NewMockPoolInterface(mockCtrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(mockCtrl)
	gitlabClient.EXPECT().
		GetAgentInfo(gomock.Any(), &agentMeta).
		Return(agentInfo, nil)

	k, cleanup, err := NewServer(ServerConfig{
		Context:                      ctx,
		Log:                          zaptest.NewLogger(t),
		GitalyPool:                   gitalyPool,
		GitLabClient:                 gitlabClient,
		AgentConfigurationPollPeriod: 10 * time.Minute,
		GitopsPollPeriod:             10 * time.Minute,
		Registerer:                   prometheus.NewPedanticRegistry(),
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	return k, agentInfo, mockCtrl, gitalyPool, gitlabClient
}
