package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"sigs.k8s.io/yaml"
)

const (
	projectId = "some/project"
	revision  = "507ebc6de9bcac25628aa7afd52802a91a0685d8"

	maxConfigurationFileSize = 128 * 1024
)

var (
	_ modserver.Module             = &module{}
	_ modserver.Factory            = &Factory{}
	_ modserver.ApplyDefaults      = ApplyDefaults
	_ rpc.AgentConfigurationServer = &server{}
)

func TestEmptyConfig(t *testing.T) {
	t.Run("comments", func(t *testing.T) {
		data := []byte(`
#gitops:
#  manifest_projects:
#  - id: "root/gitops-manifests"
#    paths:
#      - glob: "/bla/**"
`)
		assertEmpty(t, data)
	})
	t.Run("empty", func(t *testing.T) {
		data := []byte("")
		assertEmpty(t, data)
	})
	t.Run("newline", func(t *testing.T) {
		data := []byte("\n")
		assertEmpty(t, data)
	})
}

func assertEmpty(t *testing.T, data []byte) {
	config, err := parseYAMLToConfiguration(data)
	require.NoError(t, err)
	diff := cmp.Diff(config, &agentcfg.ConfigurationFile{}, protocmp.Transform())
	assert.Empty(t, diff)
}

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

func TestGetConfiguration_HappyPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, agentInfo, ctrl, gitalyPool, _ := setupServer(t)
	configFile := sampleConfig()
	resp := mock_rpc.NewMockAgentConfiguration_GetConfigurationServer(ctrl)
	resp.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	resp.EXPECT().
		Send(matcher.ProtoEq(t, &rpc.ConfigurationResponse{
			Configuration: &agentcfg.AgentConfiguration{
				Gitops: &agentcfg.GitopsCF{
					ManifestProjects: []*agentcfg.ManifestProjectCF{
						{
							Id: projectId,
						},
					},
				},
				AgentId:   agentInfo.Id,
				ProjectId: agentInfo.ProjectId,
			},
			CommitId: revision,
		}))
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	pf := mock_internalgitaly.NewMockPathFetcherInterface(ctrl)
	configFileName := agent_configuration.Directory + "/" + agentInfo.Name + "/" + agent_configuration.FileName
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &agentInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), matcher.ProtoEq(nil, agentInfo.Repository), "", gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: true,
				CommitId:        revision,
			}, nil),
		gitalyPool.EXPECT().
			PathFetcher(gomock.Any(), &agentInfo.GitalyInfo).
			Return(pf, nil),
		pf.EXPECT().
			FetchFile(gomock.Any(), matcher.ProtoEq(nil, agentInfo.Repository), []byte(revision), []byte(configFileName), int64(maxConfigurationFileSize)).
			Return(configToBytes(t, configFile), nil),
	)
	err := m.GetConfiguration(&rpc.ConfigurationRequest{
		AgentMeta: agentMeta(),
	}, resp)
	require.NoError(t, err)
}

func TestGetConfiguration_ResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, agentInfo, ctrl, gitalyPool, _ := setupServer(t)
	resp := mock_rpc.NewMockAgentConfiguration_GetConfigurationServer(ctrl)
	resp.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
		MinTimes(1)
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &agentInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), matcher.ProtoEq(nil, agentInfo.Repository), revision, gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: false,
				CommitId:        revision,
			}, nil),
	)
	err := m.GetConfiguration(&rpc.ConfigurationRequest{
		CommitId:  revision, // same commit id
		AgentMeta: agentMeta(),
	}, resp)
	require.NoError(t, err)
}

func TestGetConfiguration_UserErrors(t *testing.T) {
	gitalyErrs := []error{
		gitaly.NewNotFoundError("Bla", "some/file"),
		gitaly.NewFileTooBigError(nil, "Bla", "some/file"),
		gitaly.NewUnexpectedTreeEntryTypeError("Bla", "some/file"),
	}
	for _, gitalyErr := range gitalyErrs {
		t.Run(gitalyErr.(*gitaly.Error).Code.String(), func(t *testing.T) { // nolint: errorlint
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m, agentInfo, ctrl, gitalyPool, mockApi := setupServer(t)
			resp := mock_rpc.NewMockAgentConfiguration_GetConfigurationServer(ctrl)
			resp.EXPECT().
				Context().
				Return(mock_modserver.IncomingCtx(ctx, t, testhelpers.AgentkToken)).
				MinTimes(1)
			p := mock_internalgitaly.NewMockPollerInterface(ctrl)
			pf := mock_internalgitaly.NewMockPathFetcherInterface(ctrl)
			configFileName := agent_configuration.Directory + "/" + agentInfo.Name + "/" + agent_configuration.FileName
			gomock.InOrder(
				gitalyPool.EXPECT().
					Poller(gomock.Any(), &agentInfo.GitalyInfo).
					Return(p, nil),
				p.EXPECT().
					Poll(gomock.Any(), matcher.ProtoEq(nil, agentInfo.Repository), "", gitaly.DefaultBranch).
					Return(&gitaly.PollInfo{
						UpdateAvailable: true,
						CommitId:        revision,
					}, nil),
				gitalyPool.EXPECT().
					PathFetcher(gomock.Any(), &agentInfo.GitalyInfo).
					Return(pf, nil),
				pf.EXPECT().
					FetchFile(gomock.Any(), matcher.ProtoEq(nil, agentInfo.Repository), []byte(revision), []byte(configFileName), int64(maxConfigurationFileSize)).
					Return(nil, gitalyErr),
				mockApi.EXPECT().
					HandleProcessingError(gomock.Any(), gomock.Any(), "Config: failed to fetch",
						matcher.ErrorEq(fmt.Sprintf("agent configuration file: %v", gitalyErr)),
					),
			)
			err := m.GetConfiguration(&rpc.ConfigurationRequest{
				AgentMeta: agentMeta(),
			}, resp)
			assert.EqualError(t, err, fmt.Sprintf("rpc error: code = FailedPrecondition desc = Config: agent configuration file: %v", gitalyErr))
		})
	}
}

func setupServer(t *testing.T) (*server, *api.AgentInfo, *gomock.Controller, *mock_internalgitaly.MockPoolInterface, *mock_modserver.MockAPI) {
	ctrl := gomock.NewController(t)
	mockApi := mock_modserver.NewMockAPIWithMockPoller(ctrl, 1)
	gitalyPool := mock_internalgitaly.NewMockPoolInterface(ctrl)
	agentTracker := mock_agent_tracker.NewMockTracker(ctrl)
	gitLabClient := mock_gitlab.SetupClient(t, gapi.AgentConfigurationApiPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	s := &server{
		api:                        mockApi,
		agentRegisterer:            agentTracker,
		gitaly:                     gitalyPool,
		gitLabClient:               gitLabClient,
		getConfigurationPollConfig: testhelpers.NewPollConfig(10 * time.Minute),
		maxConfigurationFileSize:   maxConfigurationFileSize,
	}
	agentInfo := testhelpers.AgentInfoObj()
	connMatcher := matcher.ProtoEq(t, &agent_tracker.ConnectedAgentInfo{
		AgentMeta: agentMeta(),
		AgentId:   agentInfo.Id,
		ProjectId: agentInfo.ProjectId,
	}, protocmp.IgnoreFields(&agent_tracker.ConnectedAgentInfo{}, "connected_at", "connection_id"))
	gomock.InOrder(
		mockApi.EXPECT().
			GetAgentInfo(gomock.Any(), gomock.Any(), testhelpers.AgentkToken).
			Return(agentInfo, nil),
		agentTracker.EXPECT().
			RegisterConnection(gomock.Any(), connMatcher),
	)
	agentTracker.EXPECT().
		UnregisterConnection(gomock.Any(), connMatcher)
	return s, agentInfo, ctrl, gitalyPool, mockApi
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

func agentMeta() *modshared.AgentMeta {
	return &modshared.AgentMeta{
		Version:      "v1.2.3",
		CommitId:     "32452345",
		PodNamespace: "ns1",
		PodName:      "n1",
	}
}
