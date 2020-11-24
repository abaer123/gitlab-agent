package observability_agent

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient"
	"go.uber.org/zap"
)

type Factory struct {
	LogLevel zap.AtomicLevel
}

func (f *Factory) New(api modclient.AgentAPI) modclient.Module {
	return &Module{
		LogLevel: f.LogLevel,
	}
}
