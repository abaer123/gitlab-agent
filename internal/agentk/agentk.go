package agentk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

type Agent struct {
	Log                             *zap.Logger
	KasClient                       agentrpc.KasClient
	RefreshConfigurationRetryPeriod time.Duration
	ModuleFactories                 []modagent.Factory
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
	stage := st.NextStage()
	modules := make([]modagent.Module, 0, len(a.ModuleFactories))
	for _, factory := range a.ModuleFactories {
		module := factory.New(a.KasClient)
		modules = append(modules, module)
		stage.Go(module.Run)
	}
	return modules
}

func (a *Agent) startConfigurationRefresh(st stager.Stager, modules []modagent.Module) {
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		retry.JitterUntil(ctx, a.RefreshConfigurationRetryPeriod, a.refreshConfiguration(modules))
		return nil
	})
}

func (a *Agent) refreshConfiguration(modules []modagent.Module) func(context.Context) {
	var lastProcessedCommitId string
	return func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req := &agentrpc.ConfigurationRequest{
			CommitId: lastProcessedCommitId,
		}
		res, err := a.KasClient.GetConfiguration(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				a.Log.Warn("GetConfiguration failed", zap.Error(err))
			}
			return
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
				case grpctool.RequestCanceled(err):
				default:
					a.Log.Warn("GetConfiguration.Recv failed", zap.Error(err))
				}
				return
			}
			lastProcessedCommitId = config.CommitId
			err = a.applyConfiguration(modules, config.Configuration)
			if err != nil {
				a.Log.Error("Failed to apply configuration", zap.Error(err))
				continue
			}
		}
	}
}

func (a *Agent) applyConfiguration(modules []modagent.Module, config *agentcfg.AgentConfiguration) error {
	a.Log.Debug("Applying configuration", agentConfig(config))
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
