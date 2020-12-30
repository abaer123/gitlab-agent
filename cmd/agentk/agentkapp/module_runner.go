package agentkapp

import (
	"context"
	"fmt"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	agent_configuration_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

type moduleHolder struct {
	module modagent.Module
	cfg    chan *agentcfg.AgentConfiguration
}

func (h *moduleHolder) Run(ctx context.Context) error {
	return h.module.Run(ctx, h.cfg)
}

type moduleRunner struct {
	log                  *zap.Logger
	modules              []modagent.Module
	configurationWatcher agent_configuration_rpc.ConfigurationWatcherInterface
}

func (r *moduleRunner) Run(ctx context.Context) error {
	holders := make([]moduleHolder, 0, len(r.modules))
	for _, module := range r.modules {
		holders = append(holders, moduleHolder{
			module: module,
			cfg:    make(chan *agentcfg.AgentConfiguration),
		})
	}
	return cmd.RunStages(ctx,
		func(stage stager.Stage) {
			for _, holder := range holders {
				holder := holder // capture the right variable
				stage.Go(holder.Run)
			}
		},
		func(stage stager.Stage) {
			stage.Go(func(ctx context.Context) error {
				defer func() {
					for _, holder := range holders {
						close(holder.cfg)
					}
				}()
				r.configurationWatcher.Watch(ctx, func(ctx context.Context, data agent_configuration_rpc.ConfigurationData) {
					err := r.applyConfiguration(ctx, holders, data.CommitId, data.Config)
					if err != nil {
						if !errz.ContextDone(err) {
							r.log.Error("Failed to apply configuration", logz.CommitId(data.CommitId), zap.Error(err))
						}
						return
					}
				})
				return nil
			})
		},
	)
}

func (r *moduleRunner) applyConfiguration(ctx context.Context, holders []moduleHolder, commitId string, config *agentcfg.AgentConfiguration) error {
	r.log.Debug("Applying configuration", logz.CommitId(commitId), agentConfig(config))
	// Default and validate before setting for use.
	for _, holder := range holders {
		err := holder.module.DefaultAndValidateConfiguration(config)
		if err != nil {
			return fmt.Errorf("%s: %v", holder.module.Name(), err)
		}
	}
	// Set for use.
	for _, holder := range holders {
		select {
		case <-ctx.Done():
		case holder.cfg <- config:
		}
	}
	return nil
}
