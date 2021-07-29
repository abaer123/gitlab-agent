package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
)

const (
	getConfigurationInitBackoff   = 10 * time.Second
	getConfigurationMaxBackoff    = time.Minute
	getConfigurationResetDuration = time.Minute
	getConfigurationBackoffFactor = 2.0
	getConfigurationJitter        = 1.0
)

type Factory struct {
	AgentRegisterer agent_tracker.Registerer
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	agent := config.Config.Agent
	rpc.RegisterAgentConfigurationServer(config.AgentServer, &server{
		api:                      config.Api,
		gitaly:                   config.Gitaly,
		gitLabClient:             config.GitLabClient,
		agentRegisterer:          f.AgentRegisterer,
		maxConfigurationFileSize: int64(agent.Configuration.MaxConfigurationFileSize),
		getConfigurationPollConfig: retry.NewPollConfigFactory(agent.Configuration.PollPeriod.AsDuration(), retry.NewExponentialBackoffFactory(
			getConfigurationInitBackoff,
			getConfigurationMaxBackoff,
			getConfigurationResetDuration,
			getConfigurationBackoffFactor,
			getConfigurationJitter,
		)),
	})
	return &module{}, nil
}

func (f *Factory) Name() string {
	return agent_configuration.ModuleName
}
