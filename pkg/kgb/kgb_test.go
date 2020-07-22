package kgb

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/gitlab/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/protomock"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

const (
	token     = "abfaasdfasdfasdf"
	projectId = "some/project"
	revision  = "asdfadsfasdfasdfas"
)

func TestYAMLToConfigurationAndBack(t *testing.T) {
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
			given: `deployments: {}
`,
			expected: `deployments: {}
`,
		},
		{
			given: `deployments:
  manifest_projects: []
`,
			expected: `deployments: {}
`, // empty slice is omitted
		},
		{
			expected: `deployments:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
			given: `deployments:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			config, err := parseYAMLToConfiguration([]byte(tc.given))
			require.NoError(t, err)
			configJson, err := protojson.Marshal(config)
			require.NoError(t, err)
			configYaml, err := yaml.JSONToYAML(configJson)
			require.NoError(t, err)
			diff := cmp.Diff(tc.expected, string(configYaml))
			assert.Empty(t, diff)
		})
	}
}

func TestGetConfiguration(t *testing.T) {
	a, agentInfo, mockCtrl, gitalyClient, _ := setupKgb(t)
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: &agentInfo.Repository,
		Revision:   []byte("master"),
		Path:       []byte(agentConfigurationDirectory + "/" + agentInfo.Name + "/" + agentConfigurationFileName),
		Limit:      fileSizeLimit,
	}
	configFile := sampleConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockGitLabService_GetConfigurationServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
		MinTimes(1)
	resp.EXPECT().
		Send(protomock.Eq(&agentrpc.ConfigurationResponse{
			Configuration: &agentrpc.AgentConfiguration{
				Deployments: configFile.Deployments,
			},
		})).
		DoAndReturn(func(resp *agentrpc.ConfigurationResponse) error {
			cancel() // stop streaming call after the first response has been sent
			return nil
		})
	mockTreeEntry(mockCtrl, gitalyClient, treeEntryReq, configToBytes(t, configFile))
	err := a.GetConfiguration(&agentrpc.ConfigurationRequest{}, resp)
	require.NoError(t, err)
}

func TestGetObjectsToSynchronize(t *testing.T) {
	a, agentInfo, mockCtrl, gitalyClient, gitlabClient := setupKgb(t)

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
	objectsYAML := objsToYAML(t, objects...)

	projectRepo := gitalypb.Repository{
		StorageName:        "StorageName1",
		RelativePath:       "RelativePath1",
		GitObjectDirectory: "GitObjectDirectory1",
		GlRepository:       "GlRepository1",
		GlProjectPath:      "GlProjectPath1",
	}
	projectInfo := &api.ProjectInfo{
		ProjectId:  234,
		Repository: projectRepo,
	}
	findCommitReq := &gitalypb.FindCommitRequest{
		Repository: &projectRepo,
		Revision:   []byte("master"),
	}
	findCommitResp := &gitalypb.FindCommitResponse{
		Commit: &gitalypb.GitCommit{
			Id: revision,
		},
	}
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: &projectRepo,
		Revision:   []byte(findCommitResp.Commit.Id),
		Path:       []byte("manifest.yaml"),
		Limit:      fileSizeLimit,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = incomingCtx(ctx, t)
	resp := mock_agentrpc.NewMockGitLabService_GetObjectsToSynchronizeServer(mockCtrl)
	resp.EXPECT().
		Context().
		Return(ctx).
		MinTimes(1)
	resp.EXPECT().
		Send(protomock.Eq(&agentrpc.ObjectsToSynchronizeResponse{
			Revision: revision,
			Objects: []*agentrpc.ObjectToSynchronize{
				{
					Object: objectsYAML,
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

	gitalyClient.EXPECT().
		FindCommit(gomock.Any(), protomock.Eq(findCommitReq), gomock.Any()).
		Return(findCommitResp, nil)
	mockTreeEntry(mockCtrl, gitalyClient, treeEntryReq, objectsYAML)
	err := a.GetObjectsToSynchronize(&agentrpc.ObjectsToSynchronizeRequest{ProjectId: projectId}, resp)
	require.NoError(t, err)
}

func objsToYAML(t *testing.T, objs ...runtime.Object) []byte {
	out := &bytes.Buffer{}
	w := json.YAMLFramer.NewFrameWriter(out)
	for _, obj := range objs {
		data, err := yaml.Marshal(obj)
		require.NoError(t, err)
		_, err = w.Write(data)
		require.NoError(t, err)
	}
	return out.Bytes()
}

func mockTreeEntry(mockCtrl *gomock.Controller, gitalyClient *mock_gitaly.MockCommitServiceClient, req *gitalypb.TreeEntryRequest, data []byte) {
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
	gitalyClient.EXPECT().
		TreeEntry(gomock.Any(), protomock.Eq(req)).
		Return(treeEntryClient, nil)
}

func incomingCtx(ctx context.Context, t *testing.T) context.Context {
	creds := apiutil.NewTokenCredentials(token, false)
	meta, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	return metadata.NewIncomingContext(ctx, metadata.New(meta))
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
		Deployments: &agentcfg.DeploymentsCF{
			ManifestProjects: []*agentcfg.ManifestProjectCF{
				{
					Id: projectId,
				},
			},
		},
	}
}

func setupKgb(t *testing.T) (*Agent, *api.AgentInfo, *gomock.Controller, *mock_gitaly.MockCommitServiceClient, *mock_gitlab.MockGitLabClient) {
	agentMeta := api.AgentMeta{
		Token:   token,
		Version: "",
	}
	agentInfo := &api.AgentInfo{
		Meta: agentMeta,
		Id:   123,
		Name: "agent1",
		Repository: gitalypb.Repository{
			StorageName:        "StorageName",
			RelativePath:       "RelativePath",
			GitObjectDirectory: "GitObjectDirectory",
			GlRepository:       "GlRepository",
			GlProjectPath:      "GlProjectPath",
		},
	}

	mockCtrl := gomock.NewController(t)
	gitalyClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	gitlabClient := mock_gitlab.NewMockGitLabClient(mockCtrl)
	gitlabClient.EXPECT().
		GetAgentInfo(gomock.Any(), &agentMeta).
		Return(agentInfo, nil)

	a := &Agent{
		ReloadConfigurationPeriod: 10 * time.Minute,
		CommitServiceClient:       gitalyClient,
		GitLabClient:              gitlabClient,
	}

	return a, agentInfo, mockCtrl, gitalyClient, gitlabClient
}
