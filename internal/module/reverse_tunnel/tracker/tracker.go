package tracker

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/redistool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
)

type Registerer interface {
	// RegisterTunnel schedules the tunnel to be registered with the tracker.
	// Returns true on success and false if ctx signaled done.
	RegisterTunnel(ctx context.Context, info *TunnelInfo) bool
	// UnregisterTunnel schedules the tunnel to be unregistered with the tracker.
	// Returns true on success and false if ctx signaled done.
	UnregisterTunnel(ctx context.Context, info *TunnelInfo) bool
}

type Tracker interface {
	Registerer
	Run(ctx context.Context) error
}

type RedisTracker struct {
	log              *zap.Logger
	refreshPeriod    time.Duration
	gcPeriod         time.Duration
	tunnelsByAgentId redistool.ExpiringHashInterface // agentId -> connectionId -> TunnelInfo
	toRegister       chan *TunnelInfo
	toUnregister     chan *TunnelInfo
}

func NewRedisTracker(log *zap.Logger, client redis.UniversalClient, agentKeyPrefix string, ttl, refreshPeriod, gcPeriod time.Duration) *RedisTracker {
	return &RedisTracker{
		log:              log,
		refreshPeriod:    refreshPeriod,
		gcPeriod:         gcPeriod,
		tunnelsByAgentId: redistool.NewExpiringHash(log, client, tunnelsByAgentIdHashKey(agentKeyPrefix), ttl),
		toRegister:       make(chan *TunnelInfo),
		toUnregister:     make(chan *TunnelInfo),
	}
}

func (t *RedisTracker) Run(ctx context.Context) error {
	refreshTicker := time.NewTicker(t.refreshPeriod)
	defer refreshTicker.Stop()
	gcTicker := time.NewTicker(t.gcPeriod)
	defer gcTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-refreshTicker.C:
			err := t.refreshRegistrations(ctx)
			if err != nil {
				t.log.Error("Failed to refresh data in Redis", zap.Error(err))
			}
		case <-gcTicker.C:
			deletedKeys, err := t.runGc(ctx)
			if err != nil {
				t.log.Error("Failed to GC data in Redis", zap.Error(err))
				// fallthrough
			}
			if deletedKeys > 0 {
				t.log.Info("Deleted expired agent tunnel records", logz.RemovedHashKeys(deletedKeys))
			}
		case toReg := <-t.toRegister:
			err := t.registerConnection(ctx, toReg)
			if err != nil {
				t.log.Error("Failed to register tunnel", zap.Error(err))
			}
		case toUnreg := <-t.toUnregister:
			err := t.unregisterConnection(ctx, toUnreg)
			if err != nil {
				t.log.Error("Failed to unregister tunnel", zap.Error(err))
			}
		}
	}
}

func (t *RedisTracker) RegisterTunnel(ctx context.Context, info *TunnelInfo) bool {
	select {
	case <-ctx.Done():
		return false
	case t.toRegister <- info:
		return true
	}
}

func (t *RedisTracker) UnregisterTunnel(ctx context.Context, info *TunnelInfo) bool {
	select {
	case <-ctx.Done():
		return false
	case t.toUnregister <- info:
		return true
	}
}

func (t *RedisTracker) registerConnection(ctx context.Context, info *TunnelInfo) error {
	infoAny, err := anypb.New(info)
	if err != nil {
		// This should never happen
		return err
	}
	return t.tunnelsByAgentId.Set(ctx, info.AgentId, info.ConnectionId, infoAny)
}

func (t *RedisTracker) unregisterConnection(ctx context.Context, unreg *TunnelInfo) error {
	return t.tunnelsByAgentId.Unset(ctx, unreg.AgentId, unreg.ConnectionId)
}

func (t *RedisTracker) refreshRegistrations(ctx context.Context) error {
	return t.tunnelsByAgentId.Refresh(ctx)
}

func (t *RedisTracker) runGc(ctx context.Context) (int, error) {
	return t.tunnelsByAgentId.GC(ctx)
}

// tunnelsByAgentIdHashKey returns a key for agentId -> (connectionId -> marshaled TunnelInfo).
func tunnelsByAgentIdHashKey(agentKeyPrefix string) redistool.KeyToRedisKey {
	prefix := agentKeyPrefix + ":conn_by_agent_id:"
	return func(agentId interface{}) string {
		return redistool.PrefixedInt64Key(prefix, agentId.(int64))
	}
}
