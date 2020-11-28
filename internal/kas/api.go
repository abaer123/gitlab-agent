package kas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	connectionMaxAgeJitterPercent = 5
)

var (
	_ modserver.API = &API{}
)

type API struct {
	GitLabClient gitlab.ClientInterface
	ErrorTracker errortracking.Tracker
}

func (a *API) Capture(err error, opts ...errortracking.CaptureOption) {
	a.ErrorTracker.Capture(err, opts...)
}

func (a *API) GetAgentInfo(ctx context.Context, log *zap.Logger, agentMeta *api.AgentMeta, noErrorOnUnknownError bool) (*api.AgentInfo, error, bool) {
	agentInfo, err := a.GitLabClient.GetAgentInfo(ctx, agentMeta)
	switch {
	case err == nil:
		return agentInfo, nil, false
	case errz.ContextDone(err):
		err = status.Error(codes.Unavailable, "unavailable")
	case gitlab.IsForbidden(err):
		err = status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		err = status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		a.LogAndCapture(ctx, log, "GetAgentInfo()", err)
		if noErrorOnUnknownError {
			err = nil
		} else {
			err = status.Error(codes.Unavailable, "unavailable")
		}
	}
	return nil, err, true
}

func (a *API) PollImmediateUntil(ctx context.Context, interval, connectionMaxAge time.Duration, condition modserver.ConditionFunc) error {
	// this context must only be used here, not inside of condition() - connection should be closed only when idle.
	ageCtx, cancel := context.WithTimeout(ctx, mathz.DurationWithJitter(connectionMaxAge, connectionMaxAgeJitterPercent))
	defer cancel()
	err := retry.PollImmediateUntil(ageCtx, interval, wait.ConditionFunc(condition))
	if errors.Is(err, wait.ErrWaitTimeout) {
		return nil // all good, ctx is done
	}
	return err
}

func (a *API) HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error) {
	if grpctool.RequestCanceled(err) {
		// An error caused by context signalling done
		return
	}
	var ue *errz.UserError
	isUserError := errors.As(err, &ue)
	if isUserError {
		// TODO Don't log it, send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// Log at Info for now.
		log.Info(msg, zap.Error(err))
	} else {
		a.LogAndCapture(ctx, log, msg, err)
	}
}

func (a *API) HandleSendError(log *zap.Logger, msg string, err error) error {
	// The problem is almost certainly with the client's connection.
	// Still log it on Debug.
	if !grpctool.RequestCanceled(err) {
		log.Debug(msg, zap.Error(err))
	}
	return status.Error(codes.Unavailable, "gRPC send failed")
}

func (a *API) LogAndCapture(ctx context.Context, log *zap.Logger, msg string, err error) {
	// don't add logz.CorrelationIdFromContext(ctx) here as it's been added to the logger already
	log.Error(msg, zap.Error(err))
	a.Capture(fmt.Errorf("%s: %v", msg, err), errortracking.WithContext(ctx))
}
