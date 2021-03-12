package agent

import (
	"context"
	"fmt"
	"net"

	"github.com/ash2k/stager"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

type module struct {
	log      *zap.Logger
	logLevel zap.AtomicLevel
}

const (
	listenAddress         = ":8080"
	prometheusUrlPath     = "/metrics"
	livenessProbeUrlPath  = "/liveness"
	readinessProbeUrlPath = "/readiness"
)

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	return cmd.RunStages(ctx,
		func(stage stager.Stage) {
			// Listen for config changes and apply to logger
			stage.Go(func(ctx context.Context) error {
				for config := range cfg {
					err := m.setConfigurationLogging(config.Observability.Logging)
					if err != nil {
						m.log.Error("Failed to apply logging configuration", zap.Error(err))
						continue
					}
				}
				return nil
			})
			// Start metrics server
			stage.Go(func(ctx context.Context) error {
				lis, err := net.Listen("tcp", listenAddress) // nolint:gosec
				if err != nil {
					return fmt.Errorf("Observability listener failed to start: %v", err)
				}
				// Error is ignored because metricSrv.Run() closes the listener and
				// a second close always produces an error.
				defer lis.Close() //nolint:errcheck

				m.log.Info("Observability endpoint is up",
					logz.NetNetworkFromAddr(lis.Addr()),
					logz.NetAddressFromAddr(lis.Addr()),
				)

				metricSrv := observability.MetricServer{
					Name:                  m.Name(),
					Listener:              lis,
					PrometheusUrlPath:     prometheusUrlPath,
					LivenessProbeUrlPath:  livenessProbeUrlPath,
					ReadinessProbeUrlPath: readinessProbeUrlPath,
					Gatherer:              prometheus.DefaultGatherer,
					Registerer:            prometheus.DefaultRegisterer,
				}

				return metricSrv.Run(ctx)
			})
		},
	)
}

func (m *module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	prototool.NotNil(&config.Observability)
	prototool.NotNil(&config.Observability.Logging)
	err := m.defaultAndValidateLogging(config.Observability.Logging)
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
