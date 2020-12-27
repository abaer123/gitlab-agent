package agentkapp

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
)

const (
	revision1 = "rev12341234_1"
	revision2 = "rev12341234_2"
)

func TestConfigurationIsApplied(t *testing.T) {
	cfg1 := &agentcfg.AgentConfiguration{}
	cfg2 := &agentcfg.AgentConfiguration{
		Gitops: &agentcfg.GitopsCF{
			ManifestProjects: []*agentcfg.ManifestProjectCF{
				{
					Id: "bla",
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	watcher := mock_rpc.NewMockConfigurationWatcherInterface(mockCtrl)
	m := mock_modagent.NewMockModule(mockCtrl)
	m.EXPECT().
		Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	gomock.InOrder(
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg1),
		m.EXPECT().
			SetConfiguration(gomock.Any(), cfg1),
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg2),
		m.EXPECT().
			SetConfiguration(gomock.Any(), cfg2),
	)
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, callback rpc.ConfigurationCallback) {
				callback(ctx, rpc.ConfigurationData{CommitId: revision1, Config: cfg1})
				callback(ctx, rpc.ConfigurationData{CommitId: revision2, Config: cfg2})
				cancel()
			}),
	)
	a := &configRefresher{
		Log:                  zaptest.NewLogger(t),
		Modules:              []modagent.Module{m},
		ConfigurationWatcher: watcher,
	}
	err := a.Run(ctx)
	require.NoError(t, err)
}
