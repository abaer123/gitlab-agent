package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultAgentConfigurationPollPeriod               = 20 * time.Second
	defaultAgentConfigurationMaxConfigurationFileSize = 128 * 1024
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Agent)
	protodefault.NotNil(&config.Agent.Configuration)
	protodefault.NotNil(&config.Agent.Listen)

	c := config.Agent.Configuration
	protodefault.Duration(&c.PollPeriod, defaultAgentConfigurationPollPeriod)
	protodefault.Uint32(&c.MaxConfigurationFileSize, defaultAgentConfigurationMaxConfigurationFileSize)
}
