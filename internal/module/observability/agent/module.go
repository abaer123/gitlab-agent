package agent

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
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

func (m *module) SetConfiguration(ctx context.Context, config *agentcfg.AgentConfiguration) error {
	err := m.setConfigurationLogging(config.Observability.Logging)
	if err != nil {
		return fmt.Errorf("logging: %v", err)
	}
	return nil
}

func (m *module) Name() string {
	return observability.ModuleName
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
