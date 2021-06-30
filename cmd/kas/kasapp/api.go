package kasapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/cache"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxConnectionAgeJitterPercent = 5
)

type apiConfig struct {
	GitLabClient           gitlab.ClientInterface
	ErrorTracker           errortracking.Tracker
	AgentInfoCacheTtl      time.Duration
	AgentInfoCacheErrorTtl time.Duration
}

type serverAPI struct {
	cfg            apiConfig
	agentInfoCache *cache.Cache
}

func newAPI(config apiConfig) *serverAPI {
	return &serverAPI{
		cfg:            config,
		agentInfoCache: cache.New(minDuration(config.AgentInfoCacheTtl, config.AgentInfoCacheErrorTtl)),
	}
}

func (a *serverAPI) Capture(err error, opts ...errortracking.CaptureOption) {
	a.cfg.ErrorTracker.Capture(err, opts...)
}

func (a *serverAPI) GetAgentInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken) (*api.AgentInfo, error) {
	agentInfo, err := a.getAgentInfoCached(ctx, agentToken)
	switch {
	case err == nil:
		return agentInfo, nil
	case errz.ContextDone(err):
		err = status.Error(codes.Unavailable, "unavailable")
	case gitlab.IsForbidden(err):
		a.logAndCapture(ctx, log, "GetAgentInfo()", err)
		err = status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		a.logAndCapture(ctx, log, "GetAgentInfo()", err)
		err = status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		a.logAndCapture(ctx, log, "GetAgentInfo()", err)
		err = status.Error(codes.Unavailable, "unavailable")
	}
	return nil, err
}

func (a *serverAPI) PollWithBackoff(ctx context.Context, backoff retry.BackoffManager, sliding bool, maxConnectionAge, interval time.Duration, f retry.PollWithBackoffFunc) error {
	// this context must only be used here, not inside of f() - connection should be closed only when idle.
	ageCtx, cancel := context.WithTimeout(ctx, mathz.DurationWithJitter(maxConnectionAge, maxConnectionAgeJitterPercent))
	defer cancel()
	err := retry.PollWithBackoff(ageCtx, backoff, sliding, interval, f)
	if errors.Is(err, retry.ErrWaitTimeout) {
		return nil // all good, ctx is done
	}
	return err
}

func (a *serverAPI) HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error) {
	if grpctool.RequestCanceled(err) {
		// An error caused by context signalling done
		return
	}
	var ue *errz.UserError
	isUserError := errors.As(err, &ue)
	if isUserError {
		// TODO Don't log it, send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// Log at Info for now.
		log.Info(msg, logz.Error(err))
	} else {
		a.logAndCapture(ctx, log, msg, err)
	}
}

func (a *serverAPI) HandleSendError(log *zap.Logger, msg string, err error) error {
	// The problem is almost certainly with the client's connection.
	// Still log it on Debug.
	if !grpctool.RequestCanceled(err) {
		log.Debug(msg, logz.Error(err))
	}
	return status.Error(codes.Unavailable, "gRPC send failed")
}

func (a *serverAPI) logAndCapture(ctx context.Context, log *zap.Logger, msg string, err error) {
	// don't add logz.CorrelationIdFromContext(ctx) here as it's been added to the logger already
	log.Error(msg, logz.Error(err))
	a.Capture(fmt.Errorf("%s: %w", msg, err), errortracking.WithContext(ctx))
}

func (a *serverAPI) getAgentInfoCached(ctx context.Context, agentToken api.AgentToken) (*api.AgentInfo, error) {
	if a.cfg.AgentInfoCacheTtl == 0 {
		return gapi.GetAgentInfo(ctx, a.cfg.GitLabClient, agentToken)
	}
	a.agentInfoCache.EvictExpiredEntries()
	entry := a.agentInfoCache.GetOrCreateCacheEntry(agentToken)
	if !entry.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.Unlock()
	var item agentInfoCacheItem
	if entry.IsNeedRefreshLocked() {
		item.agentInfo, item.err = gapi.GetAgentInfo(ctx, a.cfg.GitLabClient, agentToken)
		var ttl time.Duration
		if item.err == nil {
			ttl = a.cfg.AgentInfoCacheTtl
		} else {
			ttl = a.cfg.AgentInfoCacheErrorTtl
		}
		entry.Item = item
		entry.Expires = time.Now().Add(ttl)
	} else {
		item = entry.Item.(agentInfoCacheItem)
	}
	return item.agentInfo, item.err
}

// agentInfoCacheItem holds cached information about an agent.
type agentInfoCacheItem struct {
	agentInfo *api.AgentInfo
	err       error
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
