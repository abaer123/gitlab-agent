package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
)

type Factory struct {
	UsageTracker usage_metrics.UsageTrackerCollector
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	return &module{
		log:                  config.Log,
		api:                  config.Api,
		usageTracker:         f.UsageTracker,
		gitLabClient:         config.GitLabClient,
		usageReportingPeriod: config.Config.Observability.UsageReportingPeriod.AsDuration(),
	}
}