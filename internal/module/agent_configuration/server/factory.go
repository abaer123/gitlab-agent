package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	m := &module{
		log:                          config.Log,
		api:                          config.Api,
		gitaly:                       config.Gitaly,
		maxConfigurationFileSize:     int64(config.Config.Agent.Limits.MaxConfigurationFileSize),
		agentConfigurationPollPeriod: config.Config.Agent.Configuration.PollPeriod.AsDuration(),
		connectionMaxAge:             config.Config.Agent.Limits.ConnectionMaxAge.AsDuration(),
	}
	rpc.RegisterAgentConfigurationServer(config.AgentServer, m)
	return m
}
