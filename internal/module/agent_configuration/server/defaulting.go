package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultAgentConfigurationPollPeriod        = 20 * time.Second
	defaultAgentLimitsMaxConfigurationFileSize = 128 * 1024
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Agent)
	a := config.Agent

	protodefault.NotNil(&a.Configuration)
	protodefault.Duration(&a.Configuration.PollPeriod, defaultAgentConfigurationPollPeriod)

	protodefault.NotNil(&a.Limits)
	protodefault.Uint32(&a.Limits.MaxConfigurationFileSize, defaultAgentLimitsMaxConfigurationFileSize)
}
