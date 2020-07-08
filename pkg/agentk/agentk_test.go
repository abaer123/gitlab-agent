package agentk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/testing/mock_engine"
)

var (
	_ GitOpsEngineFactory = &DefaultGitOpsEngineFactory{}
)

func TestRunStartsWorkersAccordingToConfiguration(t *testing.T) {
	for i, config := range testConfigurations() {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			expectedNumberOfWorkers := numberOfManifestProjects(config)
			a, mockCtrl, factory := setupAgent(t, config)
			for i := 0; i < expectedNumberOfWorkers; i++ {
				engine := mock_engine.NewMockGitOpsEngine(mockCtrl)
				engine.EXPECT().
					Run().
					Return(nil, errors.New("i'm not ok, but that's ok")).
					MinTimes(1)
				factory.EXPECT().
					New(gomock.Any()).
					Return(engine)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // give it a moment to start goroutines
			defer cancel()
			err := a.Run(ctx)
			require.NoError(t, err)
			assertWorkersMatchConfiguration(t, a, config)
		})
	}
}

func TestRunUpdatesNumberOfWorkersAccordingToConfiguration(t *testing.T) {
	t.Run("increasing order", func(t *testing.T) {
		configs := sortableConfigs(testConfigurations())
		sort.Stable(configs)
		testRunUpdatesNumberOfWorkersAccordingToConfiguration(t, configs)
	})
	t.Run("decreasing order", func(t *testing.T) {
		configs := sortableConfigs(testConfigurations())
		sort.Sort(sort.Reverse(configs))
		testRunUpdatesNumberOfWorkersAccordingToConfiguration(t, configs)
	})
}

func testRunUpdatesNumberOfWorkersAccordingToConfiguration(t *testing.T, configs []*agentrpc.AgentConfiguration) {
	a, mockCtrl, factory := setupAgent(t, configs...)
	engine := mock_engine.NewMockGitOpsEngine(mockCtrl)
	engine.EXPECT().
		Run().
		Return(nil, errors.New("i'm not ok, but that's ok")).
		AnyTimes()
	factory.EXPECT().
		New(gomock.Any()).
		Return(engine).
		AnyTimes()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // give it a moment to start goroutines
	defer cancel()
	err := a.Run(ctx)
	require.NoError(t, err)
	assertWorkersMatchConfiguration(t, a, configs[len(configs)-1])
}

func testConfigurations() []*agentrpc.AgentConfiguration {
	return []*agentrpc.AgentConfiguration{
		{},
		{
			Deployments: &agentcfg.DeploymentsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: "bla1/project1",
					},
				},
			},
		},
		{
			Deployments: &agentcfg.DeploymentsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: "bla1/project1",
					},
					{
						Id: "bla2/project2",
					},
				},
			},
		},
		{
			Deployments: &agentcfg.DeploymentsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: "bla3/project3",
					},
					{
						Id: "bla2/project2",
					},
				},
			},
		},
	}
}

func assertWorkersMatchConfiguration(t *testing.T, a *Agent, config *agentrpc.AgentConfiguration) bool {
	var projects []*agentcfg.ManifestProjectCF
	if config.Deployments != nil {
		projects = config.Deployments.ManifestProjects
	}
	if !assert.Len(t, a.workers, len(projects)) {
		return false
	}
	success := true
	for _, project := range projects {
		if !assert.Contains(t, a.workers, project.Id) {
			success = false
		}
	}
	return success
}

func setupAgent(t *testing.T, configs ...*agentrpc.AgentConfiguration) (*Agent, *gomock.Controller, *mock_engine.MockGitOpsEngineFactory) {
	mockCtrl := gomock.NewController(t)
	configStream := mock_agentrpc.NewMockGitLabService_GetConfigurationClient(mockCtrl)
	var calls []*gomock.Call
	for _, config := range configs {
		configResp := &agentrpc.ConfigurationResponse{
			Configuration: config,
		}
		calls = append(calls, configStream.EXPECT().
			Recv().
			Return(configResp, nil))
	}
	calls = append(calls,
		configStream.EXPECT().
			Recv().
			Return(nil, io.EOF))
	gomock.InOrder(calls...)
	client := mock_agentrpc.NewMockGitLabServiceClient(mockCtrl)
	client.EXPECT().
		GetConfiguration(gomock.Any(), gomock.Any()).
		Return(configStream, nil)
	factory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	return New(client, &mock_engine.ThreadSafeGitOpsEngineFactory{
		EngineFactory: factory,
	}), mockCtrl, factory
}

type sortableConfigs []*agentrpc.AgentConfiguration

func (r sortableConfigs) Len() int {
	return len(r)
}

func (r sortableConfigs) Less(i, j int) bool {
	return i < j
}

func (r sortableConfigs) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func numberOfManifestProjects(cfg *agentrpc.AgentConfiguration) int {
	if cfg.Deployments == nil {
		return 0
	}
	return len(cfg.Deployments.ManifestProjects)
}