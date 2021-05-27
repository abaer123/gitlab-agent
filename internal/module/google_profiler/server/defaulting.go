package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	prototool.NotNil(&config.Observability)
	prototool.NotNil(&config.Observability.GoogleProfiler)
}
