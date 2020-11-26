package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	return &module{
		cfg:     config.Config.Observability.GoogleProfiler,
		service: config.KasName,
		version: config.Version,
	}
}
