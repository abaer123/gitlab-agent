package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
	AgentRegisterer agent_tracker.Registerer
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	agent := config.Config.Agent
	m := &module{
		api:                          config.Api,
		gitaly:                       config.Gitaly,
		agentRegisterer:              f.AgentRegisterer,
		maxConfigurationFileSize:     int64(agent.Configuration.MaxConfigurationFileSize),
		agentConfigurationPollPeriod: agent.Configuration.PollPeriod.AsDuration(),
		maxConnectionAge:             agent.Listen.MaxConnectionAge.AsDuration(),
	}
	rpc.RegisterAgentConfigurationServer(config.AgentServer, m)
	return m, nil
}
