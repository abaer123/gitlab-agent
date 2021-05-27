package agentkapp

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
)

func agentConfig(config *agentcfg.AgentConfiguration) zap.Field {
	return zap.Reflect(logz.AgentConfig, config)
}

func featureName(feature modagent.Feature) zap.Field {
	return zap.String(logz.AgentFeatureName, modagent.KnownFeatures[feature])
}

func featureStatus(enabled bool) zap.Field {
	return zap.Bool(logz.AgentFeatureStatus, enabled)
}
