package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
)

type Factory struct {
	AgentQuerier agent_tracker.Querier
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	rpc.RegisterAgentTrackerServer(config.ApiServer, &server{
		api:          config.Api,
		agentQuerier: f.AgentQuerier,
	})
	return &module{}, nil
}

func (f *Factory) Name() string {
	return agent_tracker.ModuleName
}
