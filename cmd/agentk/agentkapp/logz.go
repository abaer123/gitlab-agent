package agentkapp

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

func agentConfig(config *agentcfg.AgentConfiguration) zap.Field {
	return zap.Reflect(logz.AgentConfig, config)
}
