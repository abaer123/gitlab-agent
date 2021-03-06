package agentkapp

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/testing/protocmp"
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
	ctrl := gomock.NewController(t)
	watcher := mock_rpc.NewMockConfigurationWatcherInterface(ctrl)
	m := mock_modagent.NewMockModule(ctrl)
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	m.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) {
			c := <-cfg
			cancel1()
			assert.Empty(t, cmp.Diff(c, cfg1, protocmp.Transform()))
			c = <-cfg
			cancel2()
			assert.Empty(t, cmp.Diff(c, cfg2, protocmp.Transform()))
			<-ctx.Done()
		})
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, callback rpc.ConfigurationCallback) {
				callback(ctx, rpc.ConfigurationData{CommitId: revision1, Config: cfg1})
				<-ctx1.Done()
				callback(ctx, rpc.ConfigurationData{CommitId: revision2, Config: cfg2})
				<-ctx2.Done()
				cancel()
			}),
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg1),
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg2),
	)
	a := newModuleRunner(zaptest.NewLogger(t), []modagent.Module{m}, watcher)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.RunModules(ctx)
	})
	g.Go(func() error {
		return a.RunConfigurationRefresh(ctx)
	})
	err := g.Wait()
	require.NoError(t, err)
}

func TestConfigurationIsSquashed(t *testing.T) {
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
	ctrl := gomock.NewController(t)
	watcher := mock_rpc.NewMockConfigurationWatcherInterface(ctrl)
	m := mock_modagent.NewMockModule(ctrl)
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	m.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) {
			<-ctx1.Done()
			c := <-cfg
			cancel()
			assert.Empty(t, cmp.Diff(c, cfg2, protocmp.Transform()))
			<-ctx.Done()
		})
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, callback rpc.ConfigurationCallback) {
				callback(ctx, rpc.ConfigurationData{CommitId: revision1, Config: cfg1})
				callback(ctx, rpc.ConfigurationData{CommitId: revision2, Config: cfg2})
				cancel1()
				<-ctx.Done()
			}),
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg1),
		m.EXPECT().
			DefaultAndValidateConfiguration(cfg2),
	)
	a := newModuleRunner(zaptest.NewLogger(t), []modagent.Module{m}, watcher)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.RunModules(ctx)
	})
	g.Go(func() error {
		return a.RunConfigurationRefresh(ctx)
	})
	err := g.Wait()
	require.NoError(t, err)
}
