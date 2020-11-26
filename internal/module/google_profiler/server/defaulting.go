package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Observability)
	protodefault.NotNil(&config.Observability.GoogleProfiler)
}
