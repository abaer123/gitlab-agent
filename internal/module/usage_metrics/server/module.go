package server

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"go.uber.org/zap"
)

type module struct {
	log                  *zap.Logger
	api                  modserver.API
	usageTracker         usage_metrics.UsageTrackerCollector
	gitLabClient         gitlab.ClientInterface
	usageReportingPeriod time.Duration
}

func (m *module) Run(ctx context.Context) error {
	if m.usageReportingPeriod == 0 {
		return nil
	}
	ticker := time.NewTicker(m.usageReportingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.sendUsageInternal(ctx); err != nil {
				if !errz.ContextDone(err) {
					m.api.HandleProcessingError(ctx, m.log, "Failed to send usage data", err)
				}
			}
		}
	}
}

func (m *module) sendUsageInternal(ctx context.Context) error {
	usageData := m.usageTracker.CloneUsageData()
	if len(usageData.Counters) == 0 {
		// No new counts
		return nil
	}
	err := gapi.SendUsagePing(ctx, m.gitLabClient, usageData.Counters)
	if err != nil {
		return err // don't wrap
	}
	// Subtract the increments we've just sent
	m.usageTracker.Subtract(usageData)
	return nil
}

func (m *module) Name() string {
	return usage_metrics.ModuleName
}
