package observability_agent

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

const (
	ModuleName = "observability"
)

type module struct {
	logLevel zap.AtomicLevel
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	protodefault.NotNil(&config.Observability)
	protodefault.NotNil(&config.Observability.Logging)
	err := m.defaultAndValidateLogging(config.Observability.Logging)
	if err != nil {
		return fmt.Errorf("logging: %v", err)
	}
	return nil
}

func (m *module) SetConfiguration(config *agentcfg.AgentConfiguration) error {
	err := m.setConfigurationLogging(config.Observability.Logging)
	if err != nil {
		return fmt.Errorf("logging: %v", err)
	}
	return nil
}

func (m *module) Name() string {
	return ModuleName
}

func (m *module) defaultAndValidateLogging(logging *agentcfg.LoggingCF) error {
	_, err := logz.LevelFromString(logging.Level.String())
	return err
}

func (m *module) setConfigurationLogging(logging *agentcfg.LoggingCF) error {
	level, err := logz.LevelFromString(logging.Level.String())
	if err != nil {
		return err
	}
	m.logLevel.SetLevel(level)
	return nil
}
