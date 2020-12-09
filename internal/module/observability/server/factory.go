package server

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
	Gatherer prometheus.Gatherer
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	return &module{
		log:        config.Log,
		cfg:        config.Config.Observability,
		gatherer:   f.Gatherer,
		registerer: config.Registerer,
		serverName: fmt.Sprintf("%s/%s/%s", config.KasName, config.Version, config.CommitId),
	}
}
