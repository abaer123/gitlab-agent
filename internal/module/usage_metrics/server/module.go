package server

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"go.uber.org/zap"
)

const (
	usagePingApiPath = "/api/v4/internal/kubernetes/usage_metrics"
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
	usageData, allZeroes := m.usageTracker.CloneUsageData()
	if allZeroes {
		// No new counts
		return nil
	}
	err := m.gitLabClient.DoJSON(ctx, http.MethodPost, usagePingApiPath, nil, "", usageData.Counters, nil)
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
