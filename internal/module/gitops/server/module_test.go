package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"

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

func TestGetObjectsToSynchronize_GetProjectInfo_Forbidden(t *testing.T) {
	m, ctrl, _, _ := setupModuleBare(t, 1, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(context.Background(), t, testhelpers.AgentkToken)).
		MinTimes(1)
	err := m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetObjectsToSynchronize_GetProjectInfo_Unauthorized(t *testing.T) {
	m, ctrl, _, _ := setupModuleBare(t, 1, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(context.Background(), t, testhelpers.AgentkToken)).
		MinTimes(1)
	err := m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGetObjectsToSynchronize_GetProjectInfo_InternalServerError(t *testing.T) {
	m, ctrl, mockApi, _ := setupModuleBare(t, 1, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mockApi.EXPECT().
		HandleProcessingError(gomock.Any(), gomock.Any(), "GetProjectInfo()", matcher.ErrorEq("error kind: 0; status: 500"))
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(context.Background(), t, testhelpers.AgentkToken)).
		MinTimes(1)
	err := m.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, server)
	require.NoError(t, err) // no error here, it keep trying for any other errors
}

func TestGetObjectsToSynchronize_HappyPath(t *testing.T) {
	ctx, cancel, a, ctrl, gitalyPool, _ := setupModule(t, 1)
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
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	gomock.InOrder(
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Header_{
					Header: &rpc.ObjectsToSynchronizeResponse_Header{
						CommitId: revision,
					},
				},
			})),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[:1],
					},
				},
			})),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[1:],
					},
				},
			})),
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Trailer_{
					Trailer: &rpc.ObjectsToSynchronizeResponse_Trailer{},
				},
			})).
			Do(func(resp *rpc.ObjectsToSynchronizeResponse) {
				cancel() // stop streaming call after the first response has been sent
			}),
	)
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	pf := mock_internalgitaly.NewMockPathFetcherInterface(ctrl)
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
			Do(func(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor gitaly.FetchVisitor) {
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

func TestGetObjectsToSynchronize_ResumeConnection(t *testing.T) {
	ctx, cancel, m, ctrl, gitalyPool, _ := setupModule(t, 1)
	projInfo := projectInfo()
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	gomock.InOrder(
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

func TestGetObjectsToSynchronize_UserErrors(t *testing.T) {
	gitalyErrs := []error{
		gitaly.NewNotFoundError("Bla", "some/file"),
		gitaly.NewFileTooBigError(nil, "Bla", "some/file"),
		gitaly.NewUnexpectedTreeEntryTypeError("Bla", "some/file"),
	}
	for _, gitalyErr := range gitalyErrs {
		t.Run(gitalyErr.(*gitaly.Error).Code.String(), func(t *testing.T) { // nolint: errorlint
			ctx, _, a, ctrl, gitalyPool, mockApi := setupModule(t, 1)

			projInfo := projectInfo()
			server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
			server.EXPECT().
				Context().
				Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
				MinTimes(1)
			server.EXPECT().
				Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
					Message: &rpc.ObjectsToSynchronizeResponse_Header_{
						Header: &rpc.ObjectsToSynchronizeResponse_Header{
							CommitId: revision,
						},
					},
				}))
			mockApi.EXPECT().
				HandleProcessingError(gomock.Any(), gomock.Any(), "GitOps: failed to get objects to synchronize",
					matcher.ErrorEq(fmt.Sprintf("manifest file: %v", gitalyErr)), // nolint: scopelint
				)
			p := mock_internalgitaly.NewMockPollerInterface(ctrl)
			pf := mock_internalgitaly.NewMockPathFetcherInterface(ctrl)
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
					Return(gitalyErr), // nolint: scopelint
			)
			err := a.GetObjectsToSynchronize(&rpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				Paths: []*agentcfg.PathCF{
					{
						Glob: defaultGitOpsManifestPathGlob,
					},
				},
			}, server)
			assert.EqualError(t, err, fmt.Sprintf("rpc error: code = FailedPrecondition desc = GitOps: failed to get objects to synchronize: manifest file: %v", gitalyErr)) // nolint: scopelint
		})
	}
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
		ctrl := gomock.NewController(t)
		server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest2.yaml",
						Data:   data,
					},
				},
			}))
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
	}{
		{
			name:              "full file name",
			glob:              "simple-path/manifest.yaml",
			expectedRepoPath:  []byte("simple-path"),
			expectedRecursive: false,
		},
		{
			name:              "empty",
			glob:              "",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
		},
		{
			name:              "simple file1",
			glob:              "*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
		},
		{
			name:              "files in directory1",
			glob:              "bla/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: false,
		},
		{
			name:              "recursive files in directory1",
			glob:              "bla/**/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
		},
		{
			name:              "all files1",
			glob:              "**/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group1",
			glob:              "[a-z]*/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group2",
			glob:              "?bla/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group3",
			glob:              "bla/?aaa/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			gotRepoPath, gotRecursive := globToGitaly(tc.glob)  // nolint: scopelint
			assert.Equal(t, tc.expectedRepoPath, gotRepoPath)   // nolint: scopelint
			assert.Equal(t, tc.expectedRecursive, gotRecursive) // nolint: scopelint
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

func setupModule(t *testing.T, pollTimes int) (context.Context, context.CancelFunc, *module, *gomock.Controller, *mock_internalgitaly.MockPoolInterface, *mock_modserver.MockAPI) {
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	m, ctrl, mockApi, gitalyPool := setupModuleBare(t, pollTimes, func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequestIsCorrect(t, r, correlationId)
		assert.Equal(t, projectId, r.URL.Query().Get(projectIdQueryParam))
		testhelpers.RespondWithJSON(t, w, projectInfoRest())
	})

	return ctx, cancel, m, ctrl, gitalyPool, mockApi
}

func setupModuleBare(t *testing.T, pollTimes int, handler func(http.ResponseWriter, *http.Request)) (*module, *gomock.Controller, *mock_modserver.MockAPI, *mock_internalgitaly.MockPoolInterface) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	gitalyPool := mock_internalgitaly.NewMockPoolInterface(ctrl)
	mockApi := mock_modserver.NewMockAPIWithMockPoller(ctrl, pollTimes)
	agentInfo := testhelpers.AgentInfoObj()
	mockApi.EXPECT().
		GetAgentInfo(gomock.Any(), gomock.Any(), testhelpers.AgentkToken, false).
		Return(agentInfo, nil, false)
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
		GitLabClient: mock_gitlab.SetupClient(t, projectInfoApiPath, handler),
		Registerer:   prometheus.NewPedanticRegistry(),
		UsageTracker: usageTracker,
		AgentServer:  grpc.NewServer(),
		ApiServer:    grpc.NewServer(),
		Gitaly:       gitalyPool,
	})
	require.NoError(t, err)
	return m.(*module), ctrl, mockApi, gitalyPool
}
