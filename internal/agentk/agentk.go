package agentk

import (
	"context"
	"fmt"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/cli-runtime/pkg/resource"
)

type Agent struct {
	Log                  *zap.Logger
	AgentMeta            *modshared.AgentMeta
	KasConn              grpc.ClientConnInterface
	K8sClientGetter      resource.RESTClientGetter
	ConfigurationWatcher rpc.ConfigurationWatcherInterface
	ModuleFactories      []modagent.Factory
}

func (a *Agent) Run(ctx context.Context) error {
	st := stager.New()
	// A stage to start all modules.
	modules := a.startModules(st)
	// Configuration refresh stage. Starts after all modules and stops before all modules are stopped.
	a.startConfigurationRefresh(st, modules)
	return st.Run(ctx)
}

func (a *Agent) startModules(st stager.Stager) []modagent.Module {
	cfg := &modagent.Config{
		Log:             a.Log,
		AgentMeta:       a.AgentMeta,
		Api:             &api{},
		K8sClientGetter: a.K8sClientGetter,
		KasConn:         a.KasConn,
	}
	stage := st.NextStage()
	modules := make([]modagent.Module, 0, len(a.ModuleFactories))
	for _, factory := range a.ModuleFactories {
		module := factory.New(cfg)
		modules = append(modules, module)
		stage.Go(module.Run)
	}
	return modules
}

func (a *Agent) startConfigurationRefresh(st stager.Stager, modules []modagent.Module) {
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		a.ConfigurationWatcher.Watch(ctx, func(ctx context.Context, data rpc.ConfigurationData) {
			err := a.applyConfiguration(modules, data.CommitId, data.Config)
			if err != nil {
				a.Log.Error("Failed to apply configuration", logz.CommitId(data.CommitId), zap.Error(err))
				return
			}
		})
		return nil
	})
}

func (a *Agent) applyConfiguration(modules []modagent.Module, commitId string, config *agentcfg.AgentConfiguration) error {
	a.Log.Debug("Applying configuration", logz.CommitId(commitId), agentConfig(config))
	// Default and validate before setting for use.
	for _, module := range modules {
		err := module.DefaultAndValidateConfiguration(config)
		if err != nil {
			return fmt.Errorf("%s: %v", module.Name(), err)
		}
	}
	// Set for use.
	for _, module := range modules {
		err := module.SetConfiguration(config)
		if err != nil {
			return fmt.Errorf("%s: %v", module.Name(), err)
		}
	}
	return nil
}
