package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultObservabilityListenAddress         = "127.0.0.1:8151"
	defaultObservabilityPrometheusUrlPath     = "/metrics"
	defaultObservabilityLivenessProbeUrlPath  = "/liveness"
	defaultObservabilityReadinessProbeUrlPath = "/readiness"
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	prototool.NotNil(&config.Observability)
	o := config.Observability

	prototool.NotNil(&o.Listen)
	prototool.String(&o.Listen.Address, defaultObservabilityListenAddress)

	prototool.NotNil(&o.Prometheus)
	prototool.String(&o.Prometheus.UrlPath, defaultObservabilityPrometheusUrlPath)

	prototool.NotNil(&o.Tracing)

	prototool.NotNil(&o.Sentry)

	prototool.NotNil(&o.Logging)

	prototool.NotNil(&o.LivenessProbe)
	prototool.String(&o.LivenessProbe.UrlPath, defaultObservabilityLivenessProbeUrlPath)

	prototool.NotNil(&o.ReadinessProbe)
	prototool.String(&o.ReadinessProbe.UrlPath, defaultObservabilityReadinessProbeUrlPath)
}
