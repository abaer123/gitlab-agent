package agent

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ modagent.Module  = &module{}
	_ modagent.Factory = &Factory{}
)

func TestStartsWorkersAccordingToConfiguration(t *testing.T) {
	for caseNum, config := range testConfigurations() {
		t.Run(fmt.Sprintf("case %d", caseNum), func(t *testing.T) {
			var wg wait.Group
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			projects := config.GetGitops().GetManifestProjects() // nolint: scopelint
			expectedNumberOfWorkers := len(projects)             // nolint: scopelint
			m, ctrl, factory := setupModule(t)
			worker := NewMockGitopsWorker(ctrl)
			for i := 0; i < expectedNumberOfWorkers; i++ {
				factory.EXPECT().
					New(matcher.ProtoEq(t, projects[i])).
					Return(worker)
			}
			worker.EXPECT().
				Run(gomock.Any()).
				Times(expectedNumberOfWorkers)
			cfg := make(chan *agentcfg.AgentConfiguration)
			wg.Start(func() {
				err := m.Run(ctx, cfg)
				assert.NoError(t, err)
			})
			require.NoError(t, m.DefaultAndValidateConfiguration(config)) // nolint: scopelint
			cfg <- config                                                 // nolint: scopelint
			close(cfg)
			cancel()
			wg.Wait()
		})
	}
}

func TestUpdatesWorkersAccordingToConfiguration(t *testing.T) {
	normalOrder := testConfigurations()
	reverseOrder := testConfigurations()
	reverse(reverseOrder)
	tests := []struct {
		name    string
		configs []*agentcfg.AgentConfiguration
	}{
		{
			name:    "normal order",
			configs: normalOrder,
		},
		{
			name:    "reverse order",
			configs: reverseOrder,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			numEngines := numUniqueProjects(tc.configs) // nolint: scopelint
			m, ctrl, factory := setupModule(t)
			var wg wait.Group
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var runCalled int64
			worker := NewMockGitopsWorker(ctrl)
			worker.EXPECT().
				Run(gomock.Any()).
				Do(func(ctx context.Context) {
					currentValue := atomic.AddInt64(&runCalled, 1)
					if currentValue == int64(numEngines) {
						cancel()
					}
					<-ctx.Done()
				}).
				Times(numEngines)
			factory.EXPECT().
				New(gomock.Any()).
				Return(worker).
				Times(numEngines)
			cfg := make(chan *agentcfg.AgentConfiguration)
			wg.Start(func() {
				err := m.Run(ctx, cfg)
				assert.NoError(t, err)
			})
			for _, config := range tc.configs { // nolint: scopelint
				require.NoError(t, m.DefaultAndValidateConfiguration(config))
				cfg <- config
			}
			close(cfg)
			wg.Wait()
		})
	}
}

func setupModule(t *testing.T) (*module, *gomock.Controller, *MockGitopsWorkerFactory) {
	ctrl := gomock.NewController(t)
	workerFactory := NewMockGitopsWorkerFactory(ctrl)
	m := &module{
		log:           zaptest.NewLogger(t),
		workerFactory: workerFactory,
	}
	return m, ctrl, workerFactory
}

func numUniqueProjects(cfgs []*agentcfg.AgentConfiguration) int {
	num := 0
	projects := make(map[string]*agentcfg.ManifestProjectCF)
	for _, config := range cfgs {
		for _, proj := range config.GetGitops().GetManifestProjects() {
			old, ok := projects[proj.Id]
			if ok {
				if !proto.Equal(old, proj) {
					projects[proj.Id] = proj
					num++
				}
			} else {
				projects[proj.Id] = proj
				num++
			}
		}
	}
	return num
}

func testConfigurations() []*agentcfg.AgentConfiguration {
	const (
		project1 = "bla1/project1"
		project2 = "bla1/project2"
		project3 = "bla3/project3"
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
						Id: project3,
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

func reverse(cfgs []*agentcfg.AgentConfiguration) {
	for i, j := 0, len(cfgs)-1; i < j; i, j = i+1, j-1 {
		cfgs[i], cfgs[j] = cfgs[j], cfgs[i]
	}
}
