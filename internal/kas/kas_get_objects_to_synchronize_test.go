package kas

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetObjectsToSynchronizeGitLabClientFailures(t *testing.T) {
	t.Parallel()
	t.Run("GetAgentInfo failures", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		k, mockCtrl, _, gitlabClient, errTracker := setupKasBare(t)
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
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}),
			errTracker.EXPECT().
				Capture(gomock.Any(), gomock.Any()).
				DoAndReturn(func(err error, opts ...errortracking.CaptureOption) {
					cancel() // exception captured, cancel the context to stop the test
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		k, mockCtrl, _, gitlabClient, errTracker := setupKasBare(t)
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
				Return(nil, &gitlab.ClientError{Kind: gitlab.ErrorKindOther, StatusCode: http.StatusInternalServerError}),
			errTracker.EXPECT().
				Capture(matcher.ErrorEq("GetProjectInfo(): error kind: 0; status: 500"), gomock.Any()).
				DoAndReturn(func(err error, opts ...errortracking.CaptureOption) {
					cancel() // exception captured, cancel the context to stop the test
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
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	gomock.InOrder(
		resp.EXPECT().
			Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
					Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
						CommitId: revision,
					},
				},
			})).
			Return(nil),
		resp.EXPECT().
			Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
					Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[:1],
					},
				},
			})).
			Return(nil),
		resp.EXPECT().
			Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
					Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest.yaml",
						Data:   objectsYAML[1:],
					},
				},
			})).
			Return(nil),
		resp.EXPECT().
			Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
					Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
				},
			})).
			DoAndReturn(func(resp *agentrpc.ObjectsToSynchronizeResponse) error {
				cancel() // stop streaming call after the first response has been sent
				return nil
			}),
	)
	gitlabClient.EXPECT().
		GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
		Return(projectInfo, nil)
	p := mock_internalgitaly.NewMockPollerInterface(mockCtrl)
	pf := mock_internalgitaly.NewMockPathFetcherInterface(mockCtrl)
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &projectInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &projectInfo.Repository, "", gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: true,
				CommitId:        revision,
			}, nil),
		gitalyPool.EXPECT().
			PathFetcher(gomock.Any(), &projectInfo.GitalyInfo).
			Return(pf, nil),
		pf.EXPECT().
			Visit(gomock.Any(), &projectInfo.Repository, []byte(revision), []byte("."), true, gomock.Any()).
			DoAndReturn(func(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor gitaly.FetchVisitor) error {
				download, maxSize, err := visitor.Entry(&gitalypb.TreeEntry{
					Path:      []byte("manifest.yaml"),
					Type:      gitalypb.TreeEntry_BLOB,
					CommitOid: manifestRevision,
				})
				require.NoError(t, err)
				assert.EqualValues(t, gitOpsManifestMaxChunkSize, maxSize)
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
	err := a.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths: []*agentcfg.PathCF{
			{
				Glob: defaultGitOpsManifestPathGlob,
			},
		},
	}, resp)
	require.NoError(t, err)

	ctxRun, cancelRun := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelRun()
	a.Run(ctxRun)
}

func TestGetObjectsToSynchronizeResumeConnection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, agentInfo, mockCtrl, gitalyPool, gitlabClient, _ := setupKas(t)
	projectInfo := projectInfo()
	resp := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(incomingCtx(ctx, t)).
		MinTimes(1)
	p := mock_internalgitaly.NewMockPollerInterface(mockCtrl)
	gomock.InOrder(
		gitlabClient.EXPECT().
			GetProjectInfo(gomock.Any(), &agentInfo.Meta, projectId).
			Return(projectInfo, nil),
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &projectInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &projectInfo.Repository, revision, gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: false,
				CommitId:        revision,
			}, nil),
	)
	err := a.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		CommitId:  revision,
	}, resp)
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
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "YML file",
			path:             "manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "JSON file",
			path:             "manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "nested YAML file",
			path:             "dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "nested YML file",
			path:             "dir/manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "nested JSON file",
			path:             "dir/manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
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
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "all files 1",
			path:             "manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
		{
			name:             "all files 2",
			path:             "dir1/manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  maxGitopsManifestFileSize,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := objectsToSynchronizeVisitor{
				glob:                   tc.glob, // nolint: scopelint
				remainingTotalFileSize: maxGitopsTotalManifestFileSize,
				fileSizeLimit:          maxGitopsManifestFileSize,
				maxNumberOfFiles:       maxGitopsNumberOfFiles,
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
			remainingTotalFileSize: maxGitopsTotalManifestFileSize,
			fileSizeLimit:          maxGitopsManifestFileSize,
			maxNumberOfFiles:       1,
		}
		download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
			Path: []byte("manifest1.yaml"),
		})
		require.NoError(t, err)
		assert.EqualValues(t, maxGitopsManifestFileSize, maxSize)
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
			fileSizeLimit:          maxGitopsManifestFileSize,
			maxNumberOfFiles:       maxGitopsNumberOfFiles,
		}
		_, err := v.StreamChunk([]byte("manifest2.yaml"), []byte("data1"))
		assert.EqualError(t, err, "rpc error: code = Internal desc = unexpected negative remaining total file size")
	})
	t.Run("blob", func(t *testing.T) {
		data := []byte("data1")
		mockCtrl := gomock.NewController(t)
		stream := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeServer(mockCtrl)
		stream.EXPECT().
			Send(matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
					Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest2.yaml",
						Data:   data,
					},
				},
			})).
			Return(nil)
		v := objectsToSynchronizeVisitor{
			stream:                 stream,
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: maxGitopsTotalManifestFileSize,
			fileSizeLimit:          maxGitopsManifestFileSize,
			maxNumberOfFiles:       maxGitopsNumberOfFiles,
		}
		done, err := v.StreamChunk([]byte("manifest2.yaml"), data)
		require.NoError(t, err)
		assert.False(t, done)
		assert.EqualValues(t, maxGitopsTotalManifestFileSize-len(data), v.remainingTotalFileSize)
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
