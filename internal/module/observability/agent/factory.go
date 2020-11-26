package agent

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"go.uber.org/zap"
)

type Factory struct {
	LogLevel zap.AtomicLevel
}

func (f *Factory) New(config *modagent.Config) modagent.Module {
	return &module{
		logLevel: f.LogLevel,
	}
}
