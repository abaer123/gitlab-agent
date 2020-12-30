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
	module      modagent.Module
	cfg2pipe    chan *agentcfg.AgentConfiguration
	pipe2module chan *agentcfg.AgentConfiguration
}

func (h *moduleHolder) runModule(ctx context.Context) error {
	return h.module.Run(ctx, h.pipe2module)
}

func (h *moduleHolder) runPipe(ctx context.Context) error {
	defer close(h.pipe2module)
	var (
		nilablePipe2module chan<- *agentcfg.AgentConfiguration
		cfgToSend          *agentcfg.AgentConfiguration
	)
	// The loop consumes the incoming items from the configuration channel (cfg2pipe) and only sends the last
	// received item to the module (pipe2module). This allows to skip configuration changes that happened while the module was handling the
	// previous configuration change.
	for {
		select {
		case <-ctx.Done(): // case #1
			return nil
		case cfgToSend = <-h.cfg2pipe: // case #2
			nilablePipe2module = h.pipe2module // enable case #3
		case nilablePipe2module <- cfgToSend: // case #3, disabled when nilablePipe2module == nil i.e. when there is nothing to send
			// config sent
			cfgToSend = nil          // help GC
			nilablePipe2module = nil // disable case #3
		}
	}
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
			module:      module,
			cfg2pipe:    make(chan *agentcfg.AgentConfiguration),
			pipe2module: make(chan *agentcfg.AgentConfiguration),
		})
	}
	return cmd.RunStages(ctx,
		func(stage stager.Stage) {
			for _, holder := range holders {
				holder := holder // capture the right variable
				stage.Go(holder.runModule)
			}
		},
		func(stage stager.Stage) {
			for _, holder := range holders {
				holder := holder // capture the right variable
				stage.Go(holder.runPipe)
			}
		},
		func(stage stager.Stage) {
			stage.Go(func(ctx context.Context) error {
				r.configurationWatcher.Watch(ctx, func(ctx context.Context, data agent_configuration_rpc.ConfigurationData) {
					err := r.applyConfiguration(holders, data.CommitId, data.Config)
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

func (r *moduleRunner) applyConfiguration(holders []moduleHolder, commitId string, config *agentcfg.AgentConfiguration) error {
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
		holder.cfg2pipe <- config
	}
	return nil
}
