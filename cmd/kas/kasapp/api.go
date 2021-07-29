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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type apiConfig struct {
	GitLabClient           gitlab.ClientInterface
	ErrorTracker           errortracking.Tracker
	AgentInfoCacheTtl      time.Duration
	AgentInfoCacheErrorTtl time.Duration
}

type serverAPI struct {
	cfg            apiConfig
	agentInfoCache *cache.CacheWithErr
}

func newAPI(config apiConfig) *serverAPI {
	return &serverAPI{
		cfg:            config,
		agentInfoCache: cache.NewWithError(config.AgentInfoCacheTtl, config.AgentInfoCacheErrorTtl),
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

func (a *serverAPI) PollWithBackoff(stream grpc.ServerStream, cfg retry.PollConfig, f retry.PollWithBackoffFunc) error {
	// this context must only be used here, not inside of f() - connection should be closed only when idle.
	ageCtx := grpctool.MaxConnectionAgeContextFromStream(stream)
	err := retry.PollWithBackoff(ageCtx, cfg, f)
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
	var ue errz.UserError
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
	agentInfo, err := a.agentInfoCache.GetItem(ctx, agentToken, func() (interface{}, error) {
		return gapi.GetAgentInfo(ctx, a.cfg.GitLabClient, agentToken)
	})
	if err != nil {
		return nil, err
	}
	return agentInfo.(*api.AgentInfo), nil
}
