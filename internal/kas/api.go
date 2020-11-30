package kas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/cache"
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

	agentInfoApiPath = "/api/v4/internal/kubernetes/agent_info"
)

type APIConfig struct {
	GitLabClient           gitlab.ClientInterface
	ErrorTracker           errortracking.Tracker
	AgentInfoCacheTtl      time.Duration
	AgentInfoCacheErrorTtl time.Duration
}

type API struct {
	cfg            APIConfig
	agentInfoCache *cache.Cache
}

func NewAPI(config APIConfig) *API {
	return &API{
		cfg:            config,
		agentInfoCache: cache.New(minDuration(config.AgentInfoCacheTtl, config.AgentInfoCacheErrorTtl)),
	}
}

func (a *API) Capture(err error, opts ...errortracking.CaptureOption) {
	a.cfg.ErrorTracker.Capture(err, opts...)
}

func (a *API) GetAgentInfo(ctx context.Context, log *zap.Logger, agentMeta *api.AgentMeta, noErrorOnUnknownError bool) (*api.AgentInfo, error, bool) {
	agentInfo, err := a.getAgentInfoCached(ctx, agentMeta)
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

func (a *API) getAgentInfoCached(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
	if a.cfg.AgentInfoCacheTtl == 0 {
		return a.getAgentInfoDirect(ctx, agentMeta)
	}
	a.agentInfoCache.EvictExpiredEntries()
	entry := a.agentInfoCache.GetOrCreateCacheEntry(agentMeta.Token)
	if !entry.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.Unlock()
	var item agentInfoCacheItem
	if entry.IsNeedRefreshLocked() {
		item.agentInfo, item.err = a.getAgentInfoDirect(ctx, agentMeta)
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

func (a *API) getAgentInfoDirect(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
	response := getAgentInfoResponse{}
	err := a.cfg.GitLabClient.DoJSON(ctx, http.MethodGet, agentInfoApiPath, nil, agentMeta, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.AgentInfo{
		Meta:       *agentMeta,
		Id:         response.AgentId,
		ProjectId:  response.ProjectId,
		Name:       response.AgentName,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

// agentInfoCacheItem holds cached information about an agent.
type agentInfoCacheItem struct {
	agentInfo *api.AgentInfo
	err       error
}

type getAgentInfoResponse struct {
	ProjectId        int64                   `json:"project_id"`
	AgentId          int64                   `json:"agent_id"`
	AgentName        string                  `json:"agent_name"`
	GitalyInfo       gitlab.GitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitlab.GitalyRepository `json:"gitaly_repository"`
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
