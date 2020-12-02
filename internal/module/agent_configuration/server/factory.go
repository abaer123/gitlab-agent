package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	agent := config.Config.Agent
	m := &module{
		log:                          config.Log,
		api:                          config.Api,
		gitaly:                       config.Gitaly,
		maxConfigurationFileSize:     int64(agent.Configuration.MaxConfigurationFileSize),
		agentConfigurationPollPeriod: agent.Configuration.PollPeriod.AsDuration(),
		maxConnectionAge:             agent.Listen.MaxConnectionAge.AsDuration(),
	}
	rpc.RegisterAgentConfigurationServer(config.AgentServer, m)
	return m
}
