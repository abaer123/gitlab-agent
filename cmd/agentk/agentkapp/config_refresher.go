package agentkapp

import (
	"context"
	"fmt"

	agent_configuration_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

type configRefresher struct {
	Log                  *zap.Logger
	Modules              []modagent.Module
	ConfigurationWatcher agent_configuration_rpc.ConfigurationWatcherInterface
}

func (r *configRefresher) Run(ctx context.Context) error {
	r.ConfigurationWatcher.Watch(ctx, func(ctx context.Context, data agent_configuration_rpc.ConfigurationData) {
		err := r.applyConfiguration(ctx, r.Modules, data.CommitId, data.Config)
		if err != nil {
			if !errz.ContextDone(err) {
				r.Log.Error("Failed to apply configuration", logz.CommitId(data.CommitId), zap.Error(err))
			}
			return
		}
	})
	return nil
}

func (r *configRefresher) applyConfiguration(ctx context.Context, modules []modagent.Module, commitId string, config *agentcfg.AgentConfiguration) error {
	r.Log.Debug("Applying configuration", logz.CommitId(commitId), agentConfig(config))
	// Default and validate before setting for use.
	for _, module := range modules {
		err := module.DefaultAndValidateConfiguration(config)
		if err != nil {
			return fmt.Errorf("%s: %v", module.Name(), err)
		}
	}
	// Set for use.
	for _, module := range modules {
		err := module.SetConfiguration(ctx, config)
		if err != nil {
			return fmt.Errorf("%s: %w", module.Name(), err) // wrap
		}
	}
	return nil
}
