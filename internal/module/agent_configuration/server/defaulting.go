package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultAgentConfigurationPollPeriod               = 20 * time.Second
	defaultAgentConfigurationMaxConfigurationFileSize = 128 * 1024
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	prototool.NotNil(&config.Agent)
	prototool.NotNil(&config.Agent.Configuration)
	prototool.NotNil(&config.Agent.Listen)

	c := config.Agent.Configuration
	prototool.Duration(&c.PollPeriod, defaultAgentConfigurationPollPeriod)
	prototool.Uint32(&c.MaxConfigurationFileSize, defaultAgentConfigurationMaxConfigurationFileSize)
}
