package server

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
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
	_ rpc.AgentConfigurationServer = &module{}
)

func TestGetConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, agentInfo, ctrl, gitalyPool := setupModule(t)
	configFile := sampleConfig()
	resp := mock_rpc.NewMockAgentConfiguration_GetConfigurationServer(ctrl)
	resp.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, mock_gitlab.AgentkToken)).
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
			},
			CommitId: revision,
		}))
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	pf := mock_internalgitaly.NewMockPathFetcherInterface(ctrl)
	configFileName := agentConfigurationDirectory + "/" + agentInfo.Name + "/" + agentConfigurationFileName
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &agentInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &agentInfo.Repository, "", gitaly.DefaultBranch).
			Return(&gitaly.PollInfo{
				UpdateAvailable: true,
				CommitId:        revision,
			}, nil),
		gitalyPool.EXPECT().
			PathFetcher(gomock.Any(), &agentInfo.GitalyInfo).
			Return(pf, nil),
		pf.EXPECT().
			FetchFile(gomock.Any(), &agentInfo.Repository, []byte(revision), []byte(configFileName), int64(maxConfigurationFileSize)).
			Return(configToBytes(t, configFile), nil),
	)
	err := m.GetConfiguration(&rpc.ConfigurationRequest{
		AgentMeta: agentMeta(),
	}, resp)
	require.NoError(t, err)
}

func TestGetConfigurationResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, agentInfo, ctrl, gitalyPool := setupModule(t)
	resp := mock_rpc.NewMockAgentConfiguration_GetConfigurationServer(ctrl)
	resp.EXPECT().
		Context().
		Return(mock_modserver.IncomingCtx(ctx, t, mock_gitlab.AgentkToken)).
		MinTimes(1)
	p := mock_internalgitaly.NewMockPollerInterface(ctrl)
	gomock.InOrder(
		gitalyPool.EXPECT().
			Poller(gomock.Any(), &agentInfo.GitalyInfo).
			Return(p, nil),
		p.EXPECT().
			Poll(gomock.Any(), &agentInfo.Repository, revision, gitaly.DefaultBranch).
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

func setupModule(t *testing.T) (*module, *api.AgentInfo, *gomock.Controller, *mock_internalgitaly.MockPoolInterface) { // nolint: unparam
	ctrl := gomock.NewController(t)
	mockApi := mock_modserver.NewMockAPIWithMockPoller(ctrl, 1)
	gitalyPool := mock_internalgitaly.NewMockPoolInterface(ctrl)
	agentTracker := mock_agent_tracker.NewMockTracker(ctrl)
	m := &module{
		api:                          mockApi,
		agentRegisterer:              agentTracker,
		gitaly:                       gitalyPool,
		maxConfigurationFileSize:     maxConfigurationFileSize,
		agentConfigurationPollPeriod: 10 * time.Minute,
	}
	agentInfo := agentInfoObj()
	connMatcher := matcher.ProtoEq(t, &agent_tracker.ConnectedAgentInfo{
		AgentMeta: agentMeta(),
		AgentId:   agentInfo.Id,
		ProjectId: agentInfo.ProjectId,
	}, protocmp.IgnoreFields(&agent_tracker.ConnectedAgentInfo{}, "connected_at", "connection_id"))
	gomock.InOrder(
		mockApi.EXPECT().
			GetAgentInfo(gomock.Any(), gomock.Any(), mock_gitlab.AgentkToken, true).
			Return(agentInfo, nil, false),
		agentTracker.EXPECT().
			RegisterConnection(gomock.Any(), connMatcher),
	)
	agentTracker.EXPECT().
		UnregisterConnection(gomock.Any(), connMatcher)
	return m, agentInfo, ctrl, gitalyPool
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

func agentMeta() *modshared.AgentMeta {
	return &modshared.AgentMeta{
		Version:      "v1.2.3",
		CommitId:     "32452345",
		PodNamespace: "ns1",
		PodName:      "n1",
	}
}
