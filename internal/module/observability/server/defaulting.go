package server

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultObservabilityUsageReportingPeriod  = 1 * time.Minute
	defaultObservabilityListenAddress         = "0.0.0.0:8151"
	defaultObservabilityPrometheusUrlPath     = "/metrics"
	defaultObservabilityLivenessProbeUrlPath  = "/liveness"
	defaultObservabilityReadinessProbeUrlPath = "/readiness"
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	protodefault.NotNil(&config.Observability)
	o := config.Observability
	protodefault.Duration(&o.UsageReportingPeriod, defaultObservabilityUsageReportingPeriod)

	protodefault.NotNil(&o.Listen)
	protodefault.String(&o.Listen.Address, defaultObservabilityListenAddress)

	protodefault.NotNil(&o.Prometheus)
	protodefault.String(&o.Prometheus.UrlPath, defaultObservabilityPrometheusUrlPath)

	protodefault.NotNil(&o.Tracing)

	protodefault.NotNil(&o.Sentry)

	protodefault.NotNil(&o.Logging)

	protodefault.NotNil(&o.LivenessProbe)
	protodefault.String(&o.LivenessProbe.UrlPath, defaultObservabilityLivenessProbeUrlPath)

	protodefault.NotNil(&o.ReadinessProbe)
	protodefault.String(&o.ReadinessProbe.UrlPath, defaultObservabilityReadinessProbeUrlPath)
}
