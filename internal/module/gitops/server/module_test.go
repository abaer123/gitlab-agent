package server

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	defaultGitOpsManifestPathGlob = "**/*.{yaml,yml,json}"

	projectId        = "some/project"
	revision         = "507ebc6de9bcac25628aa7afd52802a91a0685d8"
	manifestRevision = "7afd52802a91a0685d8507ebc6de9bcac25628aa"
)

var (
	_ modserver.Module        = &module{}
	_ modserver.Factory       = &Factory{}
	_ modserver.ApplyDefaults = ApplyDefaults
	_ rpc.GitopsServer        = &module{}
)

func TestGetObjectsToSynchronizeGetProjectInfoFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, mockCtrl, mockApi, _, gitlabClient := setupModuleBare(t, 1)
	agentInfo := agentInfoObj()
	mockApi.EXPECT().
		GetAgentInfo(gomock.Any(), gomock.Any(), testhelpers.AgentkToken, false).
		Return(agentInfo, nil, false).
		Times(3)
	mockApi.EXPECT().
		PollImmediateUntil(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, interval, connectionMaxAge time.Duration, condition modserver.ConditionFunc) error {
			done, err := condition()
			if err != nil || done {
				return err
			}
			return nil
		}).Times(2)
	expectedErr := &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}
	query := url.Values{
		projectIdQueryParam: []string{projectId},
	}
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, projectInfoApiPath, query, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindForbidden, StatusCode: http.StatusForbidden}),
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, projectInfoApiPath, query, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(&gitlab.ClientError{Kind: gitlab.ErrorKindUnauthorized, StatusCode: http.StatusUnauthorized}),
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, projectInfoApiPath, query, testhelpers.AgentkToken, nil, gomock.Any()).
			Return(expectedErr),
		mockApi.EXPECT().
			HandleProcessingError(gomock.Any(), gomock.Any(), "GetProjectInfo()", expectedErr).
			DoAndReturn(func(ctx context.Context, log *zap.Logger, msg string, err error) {
				cancel() // exception captured, cancel the context to stop the test
			}),
	)
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(mockCtrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	err := m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	err = m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	err = m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.NoError(t, err)
}

func TestGetObjectsToSynchronize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, mockCtrl, gitalyPool, gitlabClient := setupModule(t, 1)
	a.syncCount.(*mock_usage_metrics.MockCounter).EXPECT().Inc()

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
	projInfo := projectInfo()
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(mockCtrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	gomock.InOrder(
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Headers_{
					Headers: &rpc.ObjectsToSynchronizeResponse_Headers{
						CommitId: revision,
					},
				},
			})).
			Return(nil),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[:1],
					},
				},
			})).
			Return(nil),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[1:],
					},
				},
			})).
			Return(nil),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Trailers_{
					Trailers: &rpc.ObjectsToSynchronizeResponse_Trailers{},
				},
			})).
			DoAndReturn(func(resp *rpc.ObjectsToSynchronizeResponse) error {
				cancel() // stop streaming call after the first response has been sent
				return nil
			}),
	)
	query := url.Values{
		projectIdQueryParam: []string{projectId},
	}
	gitlabClient.EXPECT().
		DoJSON(gomock.Any(), http.MethodGet, projectInfoApiPath, query, testhelpers.AgentkToken, nil, gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, query url.Values, agentToken api.AgentToken, body, response interface{}) error {
			testhelpers.AssignResult(response, projectInfoRest())
			return nil
		})
	p := mock_internalgitaly.NewMockPollerInterface(mockCtrl)
	pf := mock_internalgitaly.NewMockPathFetcherInterface(mockCtrl)
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &projInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &projInfo.Repository, "", gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: true,
				CommitId:        revision,
			}, nil),
		gitalyPool.EXPECT().
			PathFetcher(gomock.Any(), &projInfo.GitalyInfo).
			Return(pf, nil),
		pf.EXPECT().
			Visit(gomock.Any(), &projInfo.Repository, []byte(revision), []byte("."), true, gomock.Any()).
			DoAndReturn(func(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor gitaly.FetchVisitor) error {
				download, maxSize, err := visitor.Entry(&gitalypb.TreeEntry{
					Path:      []byte("manifest.yaml"),
					Type:      gitalypb.TreeEntry_BLOB,
					CommitOid: manifestRevision,
				})
				require.NoError(t, err)
				assert.EqualValues(t, defaultGitopsMaxManifestFileSize, maxSize)
				assert.True(t, download)

				done, err := visitor.StreamChunk([]byte("manifest.yaml"), objectsYAML[:1])
				require.NoError(t, err)
				assert.False(t, done)
				done, err = visitor.StreamChunk([]byte("manifest.yaml"), objectsYAML[1:])
				require.NoError(t, err)
				assert.False(t, done)
				return nil
			}),
	)
	err := a.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths: []*agentcfg.PathCF{
			{
				Glob: defaultGitOpsManifestPathGlob,
			},
		},
	}, server)
	require.NoError(t, err)
}

func TestGetObjectsToSynchronizeResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, mockCtrl, gitalyPool, gitlabClient := setupModule(t, 1)
	projInfo := projectInfo()
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(mockCtrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	p := mock_internalgitaly.NewMockPollerInterface(mockCtrl)
	query := url.Values{
		projectIdQueryParam: []string{projectId},
	}
	gomock.InOrder(
		gitlabClient.EXPECT().
			DoJSON(gomock.Any(), http.MethodGet, projectInfoApiPath, query, testhelpers.AgentkToken, nil, gomock.Any()).
			DoAndReturn(func(ctx context.Context, method, path string, query url.Values, agentToken api.AgentToken, body, response interface{}) error {
				testhelpers.AssignResult(response, projectInfoRest())
				return nil
			}),
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &projInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &projInfo.Repository, revision, gitaly.DefaultBranch).
			DoAndReturn(func(ctx context.Context, repo *gitalypb.Repository, lastProcessedCommitId, refName string) (*gitaly.PollInfo, error) {
				cancel() // stop the test
				return &gitaly.PollInfo{
					UpdateAvailable: false,
					CommitId:        revision,
				}, nil
			}),
	)
	err := m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		CommitId:  revision,
	}, server)
	require.NoError(t, err)
}

func TestObjectsToSynchronizeVisitor(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		glob             string
		expectedDownload bool
		expectedMaxSize  int64
		expectedErr      string
	}{
		{
			name:             "YAML file",
			path:             "manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "YML file",
			path:             "manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "JSON file",
			path:             "manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested YAML file",
			path:             "dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested YML file",
			path:             "dir/manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested JSON file",
			path:             "dir/manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "TXT file",
			path:             "manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "nested TXT file",
			path:             "dir/manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "hidden directory",
			path:             ".dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "hidden nested directory",
			path:             "dir1/.dir2/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "invalid glob",
			path:             "dir1/manifest1.yaml",
			glob:             "**.yaml", // yes, this does not match "dir/file" names. See https://github.com/bmatcuk/doublestar/issues/48
			expectedDownload: false,
		},
		{
			name:             "no match",
			path:             "dir1/manifest1.yaml",
			glob:             "dir2/*.yml",
			expectedDownload: false,
		},
		{
			name:             "weird glob",
			path:             "manifest1.yaml",
			glob:             "**.yaml",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "all files 1",
			path:             "manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "all files 2",
			path:             "dir1/manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := objectsToSynchronizeVisitor{
				glob:                   tc.glob, // nolint: scopelint
				remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
				fileSizeLimit:          defaultGitopsMaxManifestFileSize,
				maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
			}
			download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
				Path: []byte(tc.path), // nolint: scopelint
			})
			if tc.expectedErr == "" { // nolint: scopelint
				assert.Equal(t, tc.expectedDownload, download) // nolint: scopelint
				if tc.expectedDownload {                       // nolint: scopelint
					assert.Equal(t, tc.expectedMaxSize, maxSize) // nolint: scopelint
				}
			} else {
				assert.EqualError(t, err, tc.expectedErr) // nolint: scopelint
			}
		})
	}
	t.Run("too many files", func(t *testing.T) {
		v := objectsToSynchronizeVisitor{
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       1,
		}
		download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
			Path: []byte("manifest1.yaml"),
		})
		require.NoError(t, err)
		assert.EqualValues(t, defaultGitopsMaxManifestFileSize, maxSize)
		assert.True(t, download)

		_, _, err = v.Entry(&gitalypb.TreeEntry{
			Path: []byte("manifest2.yaml"),
		})
		assert.EqualError(t, err, "maximum number of manifest files limit reached: 1")
	})
	t.Run("unexpected underflow", func(t *testing.T) {
		v := objectsToSynchronizeVisitor{
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: 1,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
		}
		_, err := v.StreamChunk([]byte("manifest2.yaml"), []byte("data1"))
		assert.EqualError(t, err, "rpc error: code = Internal desc = unexpected negative remaining total file size")
	})
	t.Run("blob", func(t *testing.T) {
		data := []byte("data1")
		mockCtrl := gomock.NewController(t)
		server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(mockCtrl)
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest2.yaml",
						Data:   data,
					},
				},
			})).
			Return(nil)
		v := objectsToSynchronizeVisitor{
			server:                 server,
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
		}
		done, err := v.StreamChunk([]byte("manifest2.yaml"), data)
		require.NoError(t, err)
		assert.False(t, done)
		assert.EqualValues(t, defaultGitopsMaxTotalManifestFileSize-len(data), v.remainingTotalFileSize)
	})
}

func TestGlobToGitaly(t *testing.T) {
	tests := []struct {
		name              string
		glob              string
		expectedRepoPath  []byte
		expectedRecursive bool
		expectedGlob      string
	}{
		{
			name:              "empty",
			glob:              "",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
			expectedGlob:      "",
		},
		{
			name:              "root",
			glob:              "/",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
			expectedGlob:      "",
		},
		{
			name:              "simple file1",
			glob:              "*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
			expectedGlob:      "*.yaml",
		},
		{
			name:              "simple file2",
			glob:              "/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
			expectedGlob:      "*.yaml",
		},
		{
			name:              "files in directory1",
			glob:              "bla/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: false,
			expectedGlob:      "*.yaml",
		},
		{
			name:              "files in directory2",
			glob:              "/bla/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: false,
			expectedGlob:      "*.yaml",
		},
		{
			name:              "recursive files in directory1",
			glob:              "bla/**/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
			expectedGlob:      "**/*.yaml",
		},
		{
			name:              "recursive files in directory2",
			glob:              "/bla/**/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
			expectedGlob:      "**/*.yaml",
		},
		{
			name:              "all files1",
			glob:              "**/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
			expectedGlob:      "**/*.yaml",
		},
		{
			name:              "all files2",
			glob:              "/**/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
			expectedGlob:      "**/*.yaml",
		},
		{
			name:              "group1",
			glob:              "/[a-z]*/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
			expectedGlob:      "[a-z]*/*.yaml",
		},
		{
			name:              "group2",
			glob:              "/?bla/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
			expectedGlob:      "?bla/*.yaml",
		},
		{
			name:              "group3",
			glob:              "/bla/?aaa/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
			expectedGlob:      "?aaa/*.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			gotRepoPath, gotRecursive, gotGlob := globToGitaly(tc.glob) // nolint: scopelint
			assert.Equal(t, tc.expectedRepoPath, gotRepoPath)           // nolint: scopelint
			assert.Equal(t, tc.expectedRecursive, gotRecursive)         // nolint: scopelint
			assert.Equal(t, tc.expectedGlob, gotGlob)                   // nolint: scopelint
		})
	}
}

func projectInfoRest() *projectInfoResponse {
	return &projectInfoResponse{
		ProjectId: 234,
		GitalyInfo: gitlab.GitalyInfo{
			Address: "127.0.0.1:321321",
			Token:   "cba",
			Features: map[string]string{
				"bla": "false",
			},
		},
		GitalyRepository: gitlab.GitalyRepository{
			StorageName:   "StorageName1",
			RelativePath:  "RelativePath1",
			GlRepository:  "GlRepository1",
			GlProjectPath: "GlProjectPath1",
		},
	}
}

func projectInfo() *api.ProjectInfo {
	rest := projectInfoRest()
	return &api.ProjectInfo{
		ProjectId:  rest.ProjectId,
		GitalyInfo: rest.GitalyInfo.ToGitalyInfo(),
		Repository: rest.GitalyRepository.ToProtoRepository(),
	}
}

func setupModule(t *testing.T, pollTimes int) (*module, *gomock.Controller, *mock_internalgitaly.MockPoolInterface, *mock_gitlab.MockClientInterface) {
	m, mockCtrl, mockApi, gitalyPool, gitlabClient := setupModuleBare(t, pollTimes)
	agentInfo := agentInfoObj()
	mockApi.EXPECT().
		GetAgentInfo(gomock.Any(), gomock.Any(), testhelpers.AgentkToken, false).
		Return(agentInfo, nil, false)

	return m, mockCtrl, gitalyPool, gitlabClient
}

func setupModuleBare(t *testing.T, pollTimes int) (*module, *gomock.Controller, *mock_modserver.MockAPI, *mock_internalgitaly.MockPoolInterface, *mock_gitlab.MockClientInterface) {
	ctrl := gomock.NewController(t)
	gitalyPool := mock_internalgitaly.NewMockPoolInterface(ctrl)
	gitlabClient := mock_gitlab.NewMockClientInterface(ctrl)
	mockApi := mock_modserver.NewMockAPIWithMockPoller(ctrl, pollTimes)
	usageTracker := mock_usage_metrics.NewMockUsageTrackerInterface(ctrl)
	usageTracker.EXPECT().
		RegisterCounter(gitopsSyncCountKnownMetric).
		Return(mock_usage_metrics.NewMockCounter(ctrl))

	f := Factory{}
	config := &kascfg.ConfigurationFile{}
	ApplyDefaults(config)
	config.Agent.Gitops.ProjectInfoCacheTtl = durationpb.New(0)
	config.Agent.Gitops.ProjectInfoCacheErrorTtl = durationpb.New(0)
	m, err := f.New(&modserver.Config{
		Log:          zaptest.NewLogger(t),
		Api:          mockApi,
		Config:       config,
		GitLabClient: gitlabClient,
		Registerer:   prometheus.NewPedanticRegistry(),
		UsageTracker: usageTracker,
		AgentServer:  grpc.NewServer(),
		ApiServer:    grpc.NewServer(),
		Gitaly:       gitalyPool,
	})
	require.NoError(t, err)
	return m.(*module), ctrl, mockApi, gitalyPool, gitlabClient
}

func agentInfoObj() *api.AgentInfo {
	return &api.AgentInfo{
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
