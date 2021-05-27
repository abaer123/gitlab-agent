package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
)

func TestStartsWorkersAccordingToConfiguration(t *testing.T) {
	for caseNum, config := range testConfigurations() {
		t.Run(fmt.Sprintf("case %d", caseNum), func(t *testing.T) {
			projects := config.GetGitops().GetManifestProjects() // nolint: scopelint
			expectedNumberOfWorkers := len(projects)             // nolint: scopelint
			wm, ctrl, factory := setupWM(t)
			worker := NewMockGitopsWorker(ctrl)
			for i := 0; i < expectedNumberOfWorkers; i++ {
				factory.EXPECT().
					New(testhelpers.AgentId, matcher.ProtoEq(t, projects[i])).
					Return(worker)
			}
			worker.EXPECT().
				Run(gomock.Any()).
				Times(expectedNumberOfWorkers)
			err := defaultAndValidateConfiguration(config) // nolint: scopelint
			require.NoError(t, err)
			err = wm.ApplyConfiguration(testhelpers.AgentId, config.Gitops) // nolint: scopelint
			require.NoError(t, err)
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
			numProjects := numUniqueProjects(tc.configs) // nolint: scopelint
			wm, ctrl, factory := setupWM(t)
			worker := NewMockGitopsWorker(ctrl)
			worker.EXPECT().
				Run(gomock.Any()).
				Do(func(ctx context.Context) {
					<-ctx.Done()
				}).
				Times(numProjects)
			factory.EXPECT().
				New(testhelpers.AgentId, gomock.Any()).
				Return(worker).
				Times(numProjects)
			for _, config := range tc.configs { // nolint: scopelint
				err := defaultAndValidateConfiguration(config)
				require.NoError(t, err)
				err = wm.ApplyConfiguration(testhelpers.AgentId, config.Gitops)
				require.NoError(t, err)
			}
		})
	}
}

func setupWM(t *testing.T) (*workerManager, *gomock.Controller, *MockGitopsWorkerFactory) {
	ctrl := gomock.NewController(t)
	workerFactory := NewMockGitopsWorkerFactory(ctrl)
	wm := newWorkerManager(zaptest.NewLogger(t), workerFactory)
	t.Cleanup(wm.stopAllWorkers)
	return wm, ctrl, workerFactory
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
		{
			AgentId: testhelpers.AgentId,
		},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: project1,
					},
				},
			},
			AgentId: testhelpers.AgentId,
		},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id:               project1,
						DefaultNamespace: "abc", // update config
					},
					{
						Id: project2,
					},
				},
			},
			AgentId: testhelpers.AgentId,
		},
		{
			Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: project3,
					},
					{
						Id:               project2,
						DefaultNamespace: "abc", // update config
					},
				},
			},
			AgentId: testhelpers.AgentId,
		},
	}
}

func reverse(cfgs []*agentcfg.AgentConfiguration) {
	for i, j := 0, len(cfgs)-1; i < j; i, j = i+1, j-1 {
		cfgs[i], cfgs[j] = cfgs[j], cfgs[i]
	}
}
