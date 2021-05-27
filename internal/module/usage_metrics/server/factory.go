package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
)

type Factory struct {
	UsageTracker usage_metrics.UsageTrackerCollector
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	return &module{
		log:                  config.Log,
		api:                  config.Api,
		usageTracker:         f.UsageTracker,
		gitLabClient:         config.GitLabClient,
		usageReportingPeriod: config.Config.Observability.UsageReportingPeriod.AsDuration(),
	}, nil
}

func (f *Factory) Name() string {
	return usage_metrics.ModuleName
}
