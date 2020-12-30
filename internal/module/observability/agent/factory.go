package agent

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability"
	"go.uber.org/zap"
)

type Factory struct {
	LogLevel zap.AtomicLevel
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	return &module{
		log:      config.Log,
		logLevel: f.LogLevel,
	}, nil
}

func (f *Factory) Name() string {
	return observability.ModuleName
}
