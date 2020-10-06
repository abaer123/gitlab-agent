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
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	_ GitOpsEngineFactory = &DefaultGitOpsEngineFactory{}
)

func TestGetConfigurationResumeConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a, mockCtrl, client, _ := setupBasicAgent(t)
	configStream1 := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
	configStream2 := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
	gomock.InOrder(
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{})).
			Return(configStream1, nil),
		configStream1.EXPECT().
			Recv().
			Return(&agentrpc.ConfigurationResponse{
				Configuration: &agentcfg.AgentConfiguration{},
				CommitId:      revision,
			}, nil),
		configStream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{
				CommitId: revision,
			})).
			Return(configStream2, nil),
		configStream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	err := a.Run(ctx)
	require.NoError(t, err)
}

func TestRunStartsWorkersAccordingToConfiguration(t *testing.T) {
	for i, config := range testConfigurations() {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			expectedNumberOfWorkers := len(config.GetGitops().GetManifestProjects()) // nolint: scopelint
			ctx, a, mockCtrl, factory := setupAgentWithConfigs(t, config)            // nolint: scopelint
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
			err := a.Run(ctx)
			require.NoError(t, err)
			assertWorkersMatchConfiguration(t, a, config) // nolint: scopelint
		})
	}
}

func TestRunUpdatesWorkersAccordingToConfiguration(t *testing.T) {
	t.Run("increasing order", func(t *testing.T) {
		configs := sortableConfigs(testConfigurations())
		sort.Stable(configs)
		testRunUpdatesWorkersAccordingToConfiguration(t, configs)
	})
	t.Run("decreasing order", func(t *testing.T) {
		configs := sortableConfigs(testConfigurations())
		sort.Sort(sort.Reverse(configs))
		testRunUpdatesWorkersAccordingToConfiguration(t, configs)
	})
}

func testRunUpdatesWorkersAccordingToConfiguration(t *testing.T, configs []*agentcfg.AgentConfiguration) {
	ctx, a, mockCtrl, factory := setupAgentWithConfigs(t, configs...)
	engine := mock_engine.NewMockGitOpsEngine(mockCtrl)
	engine.EXPECT().
		Run().
		Return(nil, errors.New("i'm not ok, but that's ok")).
		AnyTimes()
	factory.EXPECT().
		New(gomock.Any()).
		Return(engine).
		AnyTimes()
	err := a.Run(ctx)
	require.NoError(t, err)
	assertWorkersMatchConfiguration(t, a, configs[len(configs)-1])
}

func testConfigurations() []*agentcfg.AgentConfiguration {
	const (
		project1 = "bla1/project1"
		project2 = "bla1/project2"
	)
	return []*agentcfg.AgentConfiguration{
		{},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: project1,
					},
				},
			},
		},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id:                 project1,
						ResourceInclusions: defaultResourceExclusions, // update config
						ResourceExclusions: defaultResourceExclusions, // update config
					},
					{
						Id: project2,
					},
				},
			},
		},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: "bla3/project3",
					},
					{
						Id:                 project2,
						ResourceInclusions: defaultResourceExclusions, // update config
						ResourceExclusions: defaultResourceExclusions, // update config
					},
				},
			},
		},
	}
}

func assertWorkersMatchConfiguration(t *testing.T, a *Agent, config *agentcfg.AgentConfiguration) bool { // nolint: unparam
	projects := config.GetGitops().GetManifestProjects()
	if !assert.Len(t, a.workers, len(projects)) {
		return false
	}
	success := true
	for _, project := range projects {
		if !assert.Contains(t, a.workers, project.Id) {
			success = false
			continue
		}
		project = proto.Clone(project).(*agentcfg.ManifestProjectCF)
		applyDefaultsToManifestProject(project)
		success = assert.Empty(t, cmp.Diff(a.workers[project.Id].worker.projectConfiguration, project, protocmp.Transform())) || success
	}
	return success
}

func setupBasicAgent(t *testing.T) (*Agent, *gomock.Controller, *mock_agentrpc.MockKasClient, *mock_engine.MockGitOpsEngineFactory) {
	mockCtrl := gomock.NewController(t)
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
	factory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	configFlags := genericclioptions.NewTestConfigFlags()
	agent := New(client, &mock_engine.ThreadSafeGitOpsEngineFactory{
		EngineFactory: factory,
	}, configFlags)
	agent.refreshConfigurationRetryPeriod = 10 * time.Millisecond
	return agent, mockCtrl, client, factory
}

func setupAgentWithConfigs(t *testing.T, configs ...*agentcfg.AgentConfiguration) (context.Context, *Agent, *gomock.Controller, *mock_engine.MockGitOpsEngineFactory) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	agent, mockCtrl, client, factory := setupBasicAgent(t)
	configStream := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
	var calls []*gomock.Call
	for _, config := range configs {
		config = proto.Clone(config).(*agentcfg.AgentConfiguration)
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
			DoAndReturn(func() (*agentrpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}))
	gomock.InOrder(calls...)
	client.EXPECT().
		GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{})).
		Return(configStream, nil)

	return ctx, agent, mockCtrl, factory
}

type sortableConfigs []*agentcfg.AgentConfiguration

func (r sortableConfigs) Len() int {
	return len(r)
}

func (r sortableConfigs) Less(i, j int) bool {
	return i < j
}

func (r sortableConfigs) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
