package agent_tracker

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Registerer interface {
	// RegisterConnection schedules the connection to be registered with the tracker.
	// Returns true on success and false if ctx signaled done.
	RegisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool
	// UnregisterConnection schedules the connection to be unregistered with the tracker.
	// Returns true on success and false if ctx signaled done.
	UnregisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool
}

type Querier interface {
	GetConnectionsByAgentId(ctx context.Context, agentId int64) ([]*ConnectedAgentInfo, error)
	GetConnectionsByProjectId(ctx context.Context, projectId int64) ([]*ConnectedAgentInfo, error)
}

type Tracker interface {
	Registerer
	Querier
	Run(ctx context.Context) error
}

type infoHolder struct {
	// infoAny is anypb.Any wrapped around ConnectedAgentInfo.
	// This is a cache to avoid marshaling ConnectedAgentInfo on each refresh.
	infoAny *anypb.Any
}

type RedisTracker struct {
	log                    *zap.Logger
	redisClient            redis.UniversalClient
	agentKeyPrefix         string
	ttl                    time.Duration
	refreshPeriod          time.Duration
	gcPeriod               time.Duration
	connectionsByAgentId   map[int64]map[int64]infoHolder // agentId -> connectionId -> info
	connectionsByProjectId map[int64]map[int64]infoHolder // projectId -> connectionId -> info
	toRegister             chan *ConnectedAgentInfo
	toUnregister           chan *ConnectedAgentInfo
}

func NewRedisTracker(log *zap.Logger, redisClient redis.UniversalClient, agentKeyPrefix string, ttl, refreshPeriod, gcPeriod time.Duration) *RedisTracker {
	return &RedisTracker{
		log:                    log,
		redisClient:            redisClient,
		agentKeyPrefix:         agentKeyPrefix,
		ttl:                    ttl,
		refreshPeriod:          refreshPeriod,
		gcPeriod:               gcPeriod,
		connectionsByAgentId:   make(map[int64]map[int64]infoHolder),
		connectionsByProjectId: make(map[int64]map[int64]infoHolder),
		toRegister:             make(chan *ConnectedAgentInfo),
		toUnregister:           make(chan *ConnectedAgentInfo),
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
			}
			if deletedKeys > 0 {
				t.log.Info("Deleted expired agent connections records", logz.RemovedAgentConnectionRecords(deletedKeys))
			}
		case toReg := <-t.toRegister:
			err := t.registerConnection(ctx, toReg)
			if err != nil {
				t.log.Error("Failed to register connection", zap.Error(err))
			}
		case toUnreg := <-t.toUnregister:
			err := t.unregisterConnection(ctx, toUnreg)
			if err != nil {
				t.log.Error("Failed to unregister connection", zap.Error(err))
			}
		}
	}
}

func (t *RedisTracker) RegisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool {
	select {
	case <-ctx.Done():
		return false
	case t.toRegister <- info:
		return true
	}
}

func (t *RedisTracker) UnregisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool {
	select {
	case <-ctx.Done():
		return false
	case t.toUnregister <- info:
		return true
	}
}

func (t *RedisTracker) GetConnectionsByAgentId(ctx context.Context, agentId int64) ([]*ConnectedAgentInfo, error) {
	return t.getConnectionsByKey(ctx, t.connectionsByAgentIdHashKey(agentId))
}

func (t *RedisTracker) GetConnectionsByProjectId(ctx context.Context, projectId int64) ([]*ConnectedAgentInfo, error) {
	return t.getConnectionsByKey(ctx, t.connectionsByProjectIdHashKey(projectId))
}

func (t *RedisTracker) getConnectionsByKey(ctx context.Context, key string) ([]*ConnectedAgentInfo, error) {
	var result []*ConnectedAgentInfo
	_, err := t.scanExpiringHash(ctx, key, func(value *anypb.Any) error {
		var info ConnectedAgentInfo
		err := value.UnmarshalTo(&info)
		if err != nil {
			return err
		}
		result = append(result, &info)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *RedisTracker) registerConnection(ctx context.Context, info *ConnectedAgentInfo) error {
	infoAny, err := anypb.New(info)
	if err != nil {
		// This should never happen
		return err
	}
	holder := infoHolder{
		infoAny: infoAny,
	}
	addToMap(t.connectionsByProjectId, info.ProjectId, info.ConnectionId, holder)
	addToMap(t.connectionsByAgentId, info.AgentId, info.ConnectionId, holder)
	data := map[int64]infoHolder{
		info.ConnectionId: holder,
	}
	err = t.refreshConnectionsByProjectId(ctx, info.ProjectId, data)
	if err != nil {
		return err
	}
	err = t.refreshConnectionsByAgentId(ctx, info.AgentId, data)
	if err != nil {
		return err
	}
	return nil
}

func (t *RedisTracker) unregisterConnection(ctx context.Context, unreg *ConnectedAgentInfo) error {
	removeFromMap(t.connectionsByAgentId, unreg.AgentId, unreg.ConnectionId)
	removeFromMap(t.connectionsByProjectId, unreg.ProjectId, unreg.ConnectionId)
	hKey := strconv.FormatInt(unreg.ConnectionId, 10)
	_, err := t.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.HDel(ctx, t.connectionsByAgentIdHashKey(unreg.AgentId), hKey)
		p.HDel(ctx, t.connectionsByProjectIdHashKey(unreg.ProjectId), hKey)
		return nil
	})
	return err
}

func (t *RedisTracker) refreshRegistrations(ctx context.Context) error {
	for agentId, data := range t.connectionsByAgentId {
		err := t.refreshConnectionsByAgentId(ctx, agentId, data)
		if err != nil {
			return err
		}
	}
	for projectId, data := range t.connectionsByProjectId {
		err := t.refreshConnectionsByProjectId(ctx, projectId, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *RedisTracker) refreshConnectionsByAgentId(ctx context.Context, agentId int64, data map[int64]infoHolder) error {
	return t.refreshHash(ctx, t.connectionsByAgentIdHashKey(agentId), data)
}

func (t *RedisTracker) refreshConnectionsByProjectId(ctx context.Context, projectId int64, data map[int64]infoHolder) error {
	return t.refreshHash(ctx, t.connectionsByProjectIdHashKey(projectId), data)
}

func (t *RedisTracker) refreshHash(ctx context.Context, key string, data map[int64]infoHolder) error {
	args := make([]interface{}, 0, len(data)*2)
	expiresAt := timestamppb.New(time.Now().Add(t.ttl))
	for connectionId, holder := range data {
		connInfo, err := proto.Marshal(&ExpiringValue{
			ExpiresAt: expiresAt,
			Value:     holder.infoAny,
		})
		if err != nil {
			// This should never happen
			t.log.Error("Failed to marshal ExpiringValue", zap.Error(err))
			continue
		}
		args = append(args, connectionId, connInfo)
	}
	_, err := t.redisClient.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, key, args)
		p.PExpire(ctx, key, t.ttl)
		return nil
	})
	return err
}

// connectionsByAgentIdHashKey returns a key for agentId -> (connectionId -> marshaled ConnectedAgentInfo).
func (t *RedisTracker) connectionsByAgentIdHashKey(agentId int64) string {
	var b strings.Builder
	b.WriteString(t.agentKeyPrefix)
	b.WriteString(":conn_by_agent_id:")
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, uint64(agentId))
	b.Write(id)
	return b.String()
}

// connectionsByProjectIdHashKey returns a key for projectId -> (agentId ->marshaled ConnectedAgentInfo).
func (t *RedisTracker) connectionsByProjectIdHashKey(projectId int64) string {
	var b strings.Builder
	b.WriteString(t.agentKeyPrefix)
	b.WriteString(":conn_by_project_id:")
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, uint64(projectId))
	b.Write(id)
	return b.String()
}

// runGc iterates all relevant stored data and deletes expired entries.
// It returns number of deleted Redis (hash) keys, including when an error occurs.
// Here we only inspect/GC hashes where we have entries. Other kas instances GC same and/or other corresponding hashes.
// Hashes that don't have a corresponding kas (e.g. because it crashed) will expire because of TTL on the hash key.
func (t *RedisTracker) runGc(ctx context.Context) (int, error) {
	var deletedKeys int
	for agentId := range t.connectionsByAgentId {
		deleted, err := t.gcHash(ctx, t.connectionsByAgentIdHashKey(agentId))
		if err != nil {
			return deletedKeys, err
		}
		deletedKeys += deleted
	}
	for projectId := range t.connectionsByProjectId {
		deleted, err := t.gcHash(ctx, t.connectionsByProjectIdHashKey(projectId))
		if err != nil {
			return deletedKeys, err
		}
		deletedKeys += deleted
	}
	return deletedKeys, nil
}

// gcHash iterates a hash and removes all expired values.
// It assumes that values are marshaled ExpiringValue.
func (t *RedisTracker) gcHash(ctx context.Context, key string) (int, error) {
	return t.scanExpiringHash(ctx, key, func(value *anypb.Any) error {
		// nothing to do
		return nil
	})
}

func (t *RedisTracker) scanExpiringHash(ctx context.Context, key string, cb func(value *anypb.Any) error) (keysDeleted int, retErr error) {
	var keysToDelete []string
	defer func() {
		if len(keysToDelete) == 0 {
			return
		}
		_, err := t.redisClient.HDel(ctx, key, keysToDelete...).Result()
		if err != nil {
			if retErr == nil {
				retErr = err
			}
			return
		}
		keysDeleted = len(keysToDelete)
	}()
	// Scan keys of a hash. See https://redis.io/commands/scan
	iter := t.redisClient.HScan(ctx, key, 0, "", 0).Iterator()
	for iter.Next(ctx) {
		k := iter.Val()
		if !iter.Next(ctx) {
			// This shouldn't happen
			return 0, errors.New("invalid Redis reply")
		}
		v := iter.Val()
		var msg ExpiringValue
		err := proto.Unmarshal([]byte(v), &msg)
		if err != nil {
			t.log.Error("Failed to unmarshal hash value", zap.Error(err), logz.RedisKey([]byte(k)))
			continue // try to skip and continue
		}
		if msg.ExpiresAt != nil && msg.ExpiresAt.AsTime().Before(time.Now()) {
			keysToDelete = append(keysToDelete, k)
			continue
		}
		err = cb(msg.Value)
		if err != nil {
			return 0, err
		}
	}
	return 0, iter.Err()
}

func addToMap(m map[int64]map[int64]infoHolder, key1, key2 int64, val infoHolder) {
	nm := m[key1]
	if nm == nil {
		nm = make(map[int64]infoHolder, 1)
		m[key1] = nm
	}
	nm[key2] = val
}

func removeFromMap(m map[int64]map[int64]infoHolder, key1, key2 int64) {
	nm := m[key1]
	delete(nm, key2)
	if len(nm) == 0 {
		delete(m, key1)
	}
}
