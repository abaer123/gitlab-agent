package kas

import (
	"context"
	"sync/atomic"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
)

func (s *Server) sendUsage(ctx context.Context) {
	if s.usageReportingPeriod == 0 {
		return
	}
	ticker := time.NewTicker(s.usageReportingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sendUsageInternal(ctx); err != nil {
				if !errz.ContextDone(err) {
					s.api.LogAndCapture(ctx, s.log, "Failed to send usage data", err)
				}
			}
		}
	}
}

func (s *Server) sendUsageInternal(ctx context.Context) error {
	m := s.usageMetrics.Clone()
	if m.IsEmptyNotThreadSafe() {
		// No new counts
		return nil
	}
	err := s.gitLabClient.SendUsage(ctx, &gitlab.UsageData{
		GitopsSyncCount: m.gitopsSyncCount,
	})
	if err != nil {
		return err // don't wrap
	}
	// Subtract the increments we've just sent
	s.usageMetrics.Subtract(m)
	return nil
}

type usageMetrics struct {
	gitopsSyncCount int64
}

func (m *usageMetrics) IsEmptyNotThreadSafe() bool {
	return m.gitopsSyncCount == 0
}

func (m *usageMetrics) IncGitopsSyncCount() {
	atomic.AddInt64(&m.gitopsSyncCount, 1)
}

func (m *usageMetrics) Clone() *usageMetrics {
	return &usageMetrics{
		gitopsSyncCount: atomic.LoadInt64(&m.gitopsSyncCount),
	}
}

func (m *usageMetrics) Subtract(other *usageMetrics) {
	atomic.AddInt64(&m.gitopsSyncCount, -other.gitopsSyncCount)
}
