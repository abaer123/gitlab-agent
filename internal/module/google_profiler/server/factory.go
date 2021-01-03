package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/google_profiler"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	return &module{
		cfg:     config.Config.Observability.GoogleProfiler,
		service: config.KasName,
		version: config.Version,
	}, nil
}

func (f *Factory) Name() string {
	return google_profiler.ModuleName
}
