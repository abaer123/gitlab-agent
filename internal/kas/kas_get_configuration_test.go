package kas

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
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
		Limit:      maxConfigurationFileSize,
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
				Gitops: &agentcfg.GitopsCF{
					ManifestProjects: []*agentcfg.ManifestProjectCF{
						{
							Id:               projectId,
							DefaultNamespace: defaultGitOpsManifestNamespace,
							Paths: []*agentcfg.PathCF{
								{
									Glob: defaultGitOpsManifestPathGlob,
								},
							},
						},
					},
				},
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
	ctx, cancel := context.WithCancel(context.Background())
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
