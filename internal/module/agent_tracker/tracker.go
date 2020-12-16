package agent_tracker

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/redis"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Tracker interface {
	Run(ctx context.Context) error
	// RegisterConnection schedules the connection to be registered with the tracker.
	// Returns true on success and false if ctx signaled done.
	RegisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool
	// UnregisterConnection schedules the connection to be unregistered with the tracker.
	// Returns true on success and false if ctx signaled done.
	UnregisterConnection(ctx context.Context, info *ConnectedAgentInfo) bool
}

type infoHolder struct {
	// infoAny is anypb.Any wrapped around ConnectedAgentInfo.
	// This is a cache to avoid marshaling ConnectedAgentInfo on each refresh.
	infoAny *anypb.Any
}

type RedisTracker struct {
	log                    *zap.Logger
	redis                  redis.Pool
	agentKeyPrefix         string
	ttl                    time.Duration
	ttlInMilliseconds      int64
	refreshPeriod          time.Duration
	gcPeriod               time.Duration
	connectionsByAgentId   map[int64]map[int64]infoHolder // agentId -> connectionId -> info
	connectionsByProjectId map[int64]map[int64]infoHolder // projectId -> connectionId -> info
	toRegister             chan *ConnectedAgentInfo
	toUnregister           chan *ConnectedAgentInfo
}

func NewRedisTracker(log *zap.Logger, redis redis.Pool, agentKeyPrefix string, ttl, refreshPeriod, gcPeriod time.Duration) *RedisTracker {
	return &RedisTracker{
		log:                    log,
		redis:                  redis,
		agentKeyPrefix:         agentKeyPrefix,
		ttl:                    ttl,
		ttlInMilliseconds:      int64(ttl / time.Millisecond),
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
	addToMap(t.connectionsByAgentId, info.Id, info.ConnectionId, holder)
	data := map[int64]infoHolder{
		info.ConnectionId: holder,
	}
	return t.withConn(ctx, func(conn redigo.Conn) error {
		err = t.refreshConnectionsByProjectId(conn, info.ProjectId, data)
		if err != nil {
			return err
		}
		err = t.refreshConnectionsByAgentId(conn, info.Id, data)
		if err != nil {
			return err
		}
		return nil
	})
}

func (t *RedisTracker) unregisterConnection(ctx context.Context, unreg *ConnectedAgentInfo) error {
	removeFromMap(t.connectionsByAgentId, unreg.Id, unreg.ConnectionId)
	removeFromMap(t.connectionsByProjectId, unreg.ProjectId, unreg.ConnectionId)
	return t.withConn(ctx, func(conn redigo.Conn) error {
		err := conn.Send("HDEL", t.connectionsByAgentIdHashKey(unreg.Id), unreg.ConnectionId)
		if err != nil {
			return err
		}
		err = conn.Send("HDEL", t.connectionsByProjectIdHashKey(unreg.ProjectId), unreg.ConnectionId)
		if err != nil {
			return err
		}
		return nil
	})
}

func (t *RedisTracker) refreshRegistrations(ctx context.Context) error {
	return t.withConn(ctx, func(conn redigo.Conn) error {
		for agentId, data := range t.connectionsByAgentId {
			err := t.refreshConnectionsByAgentId(conn, agentId, data)
			if err != nil {
				return err
			}
		}
		for projectId, data := range t.connectionsByProjectId {
			err := t.refreshConnectionsByProjectId(conn, projectId, data)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (t *RedisTracker) refreshConnectionsByAgentId(conn redigo.Conn, agentId int64, data map[int64]infoHolder) error {
	return t.refreshHash(conn, t.connectionsByAgentIdHashKey(agentId), data)
}

func (t *RedisTracker) refreshConnectionsByProjectId(conn redigo.Conn, projectId int64, data map[int64]infoHolder) error {
	return t.refreshHash(conn, t.connectionsByProjectIdHashKey(projectId), data)
}

func (t *RedisTracker) refreshHash(conn redigo.Conn, key []byte, data map[int64]infoHolder) error {
	args := make([]interface{}, 0, len(data)*2+1)
	args = append(args, key)
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
	err := conn.Send("MULTI")
	if err != nil {
		return err
	}
	err = conn.Send("HSET", args...)
	if err != nil {
		return err
	}
	err = conn.Send("PEXPIRE", key, t.ttlInMilliseconds)
	if err != nil {
		return err
	}
	err = conn.Send("EXEC")
	if err != nil {
		return err
	}
	return nil
}

// connectionsByAgentIdHashKey returns a key for agentId -> (connectionId -> marshaled ConnectedAgentInfo).
func (t *RedisTracker) connectionsByAgentIdHashKey(agentId int64) []byte {
	var b bytes.Buffer
	b.WriteString(t.agentKeyPrefix)
	b.WriteString(":conn_by_agent_id:")
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, uint64(agentId))
	b.Write(id)
	return b.Bytes()
}

// connectionsByProjectIdHashKey returns a key for projectId -> (agentId ->marshaled ConnectedAgentInfo).
func (t *RedisTracker) connectionsByProjectIdHashKey(projectId int64) []byte {
	var b bytes.Buffer
	b.WriteString(t.agentKeyPrefix)
	b.WriteString(":conn_by_project_id:")
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, uint64(projectId))
	b.Write(id)
	return b.Bytes()
}

// runGc iterates all relevant stored data and deletes expired entries.
// It returns number of deleted Redis (hash) keys, including when an error occurs.
func (t *RedisTracker) runGc(ctx context.Context) (int, error) {
	var deletedKeys int
	err := t.withConn(ctx, func(conn redigo.Conn) error {
		for agentId := range t.connectionsByAgentId {
			deleted, err := t.gcHash(conn, t.connectionsByAgentIdHashKey(agentId))
			if err != nil {
				return err
			}
			deletedKeys += deleted
		}
		for projectId := range t.connectionsByProjectId {
			deleted, err := t.gcHash(conn, t.connectionsByProjectIdHashKey(projectId))
			if err != nil {
				return err
			}
			deletedKeys += deleted
		}
		return nil
	})
	return deletedKeys, err
}

// gcHash iterates a hash and removes all expired values.
// It assumes that values are marshaled ExpiringValue.
func (t *RedisTracker) gcHash(conn redigo.Conn, key []byte) (int, error) {
	keysToDelete := []interface{}{key}
	// Scan keys of a hash. See https://redis.io/commands/scan
	cursor := []byte{'0'}
	for {
		reply, err := conn.Do("HSCAN", key, cursor)
		if err != nil {
			return 0, err
		}
		// See https://redis.io/commands/scan#return-value
		var pairs []interface{}
		_, err = redigo.Scan(reply.([]interface{}), &cursor, &pairs)
		if err != nil {
			return 0, err
		}
		for len(pairs) > 0 {
			var k, v []byte
			pairs, err = redigo.Scan(pairs, &k, &v)
			if err != nil {
				return 0, err
			}
			var msg ExpiringValue
			err = proto.Unmarshal(v, &msg)
			if err != nil {
				t.log.Error("Failed to unmarshal hash value", zap.Error(err))
				continue // try to skip and continue
			}
			if msg.ExpiresAt != nil && msg.ExpiresAt.AsTime().Before(time.Now()) {
				keysToDelete = append(keysToDelete, k)
			}
		}
		if bytes.Equal(cursor, []byte{'0'}) {
			break // HSCAN finished
		}
	}
	if len(keysToDelete) == 1 { // 1 element is the key of the hash which we added above
		return 0, nil
	}
	_, err := conn.Do("HDEL", keysToDelete...)
	if err != nil {
		return 0, err
	}
	return len(keysToDelete) - 1, nil
}

func (t *RedisTracker) withConn(ctx context.Context, f func(redigo.Conn) error) (retErr error) {
	return withConn(ctx, t.redis, f)
}

func withConn(ctx context.Context, pool redis.Pool, f func(redigo.Conn) error) (retErr error) {
	conn, err := pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer errz.SafeClose(conn, &retErr)
	err = f(conn)
	if err != nil {
		return err
	}
	_, err = conn.Do("") // Flush all sent commands and discard replies
	if err != nil {
		return err
	}
	return nil
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
