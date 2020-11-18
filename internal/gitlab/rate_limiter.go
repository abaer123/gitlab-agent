package gitlab

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
)

// Limiter defines the interface to perform client-side request rate limiting.
// You can use golang.org/x/time/rate.Limiter as an implementation of this interface.
type Limiter interface {
	// Wait blocks until limiter permits an event to happen.
	// It returns an error if the Context is
	// canceled, or the expected wait time exceeds the Context's Deadline.
	Wait(context.Context) error
}

type RateLimitingClient struct {
	Delegate ClientInterface
	Limiter  Limiter
}

func (r *RateLimitingClient) GetAgentInfo(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
	if err := r.Limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return r.Delegate.GetAgentInfo(ctx, agentMeta)
}

func (r *RateLimitingClient) GetProjectInfo(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
	if err := r.Limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return r.Delegate.GetProjectInfo(ctx, agentMeta, projectId)
}

func (r *RateLimitingClient) SendUsage(ctx context.Context, data *UsageData) error {
	if err := r.Limiter.Wait(ctx); err != nil {
		return err
	}
	return r.Delegate.SendUsage(ctx, data)
}
