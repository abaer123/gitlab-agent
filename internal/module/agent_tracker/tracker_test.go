package agent_tracker

import (
	"context"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/redis"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	redisURLEnvName = "REDIS_URL"
)

var (
	_ Tracker = &RedisTracker{}
)

func TestRegisterConnection(t *testing.T) {
	p, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))

	// Then
	equalHash(t, p, tr.connectionsByProjectIdHashKey(info.ProjectId), info)
	equalHash(t, p, tr.connectionsByAgentIdHashKey(info.Id), info)
}

func TestUnregisterConnection(t *testing.T) {
	p, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	require.NoError(t, tr.unregisterConnection(context.Background(), info))

	// Then
	require.Empty(t, getHash(t, p, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, p, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestHashExpires(t *testing.T) {
	t.Parallel()
	p, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	time.Sleep(tr.ttl + 100*time.Millisecond)

	// Then
	require.Empty(t, getHash(t, p, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, p, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestGC(t *testing.T) {
	t.Parallel()
	p, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	require.NoError(t, withConn(context.Background(), p, func(conn redigo.Conn) error {
		newExpireInMillis := 3 * int64(tr.ttl/time.Millisecond)
		require.NoError(t, conn.Send("PEXPIRE", tr.connectionsByProjectIdHashKey(info.ProjectId), newExpireInMillis))
		return conn.Send("PEXPIRE", tr.connectionsByAgentIdHashKey(info.Id), newExpireInMillis)
	}))
	time.Sleep(tr.ttl + 100*time.Millisecond)
	deletedKeys, err := tr.runGc(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 2, deletedKeys)

	// Then
	require.Empty(t, getHash(t, p, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, p, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestRefresh(t *testing.T) {
	p, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	registrationTime := time.Now()
	oldExpireIn := tr.ttl
	tr.ttl = 2 * tr.ttl // increase
	tr.ttlInMilliseconds = int64(tr.ttl / time.Millisecond)
	err := tr.refreshRegistrations(context.Background())
	require.NoError(t, err)

	// Then
	expireAfter := registrationTime.Add(oldExpireIn)
	valuesExpireAfter(t, p, tr.connectionsByProjectIdHashKey(info.ProjectId), expireAfter)
	valuesExpireAfter(t, p, tr.connectionsByAgentIdHashKey(info.Id), expireAfter)
}

func setupTracker(t *testing.T) (*redigo.Pool, *RedisTracker) {
	p := pool(t)
	t.Cleanup(func() {
		assert.NoError(t, p.Close())
	})
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := make([]byte, 32)
	_, err := r.Read(prefix)
	require.NoError(t, err)
	tr := NewRedisTracker(zaptest.NewLogger(t), p, string(prefix), time.Second, time.Minute, time.Minute)
	return p, tr
}

func connInfo() *ConnectedAgentInfo {
	return &ConnectedAgentInfo{
		AgentMeta: &modshared.AgentMeta{
			Version:      "v1.2.3",
			CommitId:     "123123",
			PodNamespace: "ns",
			PodName:      "name",
		},
		ConnectedAt:  timestamppb.Now(),
		ConnectionId: 123,
		Id:           345,
		ProjectId:    456,
	}
}

func pool(t *testing.T) *redigo.Pool {
	redisURL := os.Getenv(redisURLEnvName)
	if redisURL == "" {
		t.Skipf("%s environment variable not set, skipping test", redisURLEnvName)
	}
	u, err := url.Parse(redisURL)
	require.NoError(t, err)
	return redis.NewPool(&redis.Config{
		URL:          u,
		MaxActive:    1,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		KeepAlive:    time.Minute,
	})
}

func getHash(t *testing.T, p *redigo.Pool, key []byte) map[string][]byte {
	var reply interface{}
	err := withConn(context.Background(), p, func(conn redigo.Conn) error {
		var err error
		reply, err = conn.Do("HGETALL", key)
		return err
	})
	require.NoError(t, err)
	pairs := reply.([]interface{})
	result := make(map[string][]byte)
	for len(pairs) > 0 {
		var (
			k string
			v []byte
		)
		pairs, err = redigo.Scan(pairs, &k, &v)
		require.NoError(t, err)
		result[k] = v
	}
	return result
}

func equalHash(t *testing.T, p *redigo.Pool, key []byte, info *ConnectedAgentInfo) {
	hash := getHash(t, p, key)
	require.Len(t, hash, 1)
	connectionIdStr := strconv.Itoa(int(info.ConnectionId))
	require.Contains(t, hash, connectionIdStr)
	val := hash[connectionIdStr]
	var msg ExpiringValue
	err := proto.Unmarshal(val, &msg)
	require.NoError(t, err)
	var valProto ConnectedAgentInfo
	err = anypb.UnmarshalTo(msg.Value, &valProto, proto.UnmarshalOptions{})
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(info, &valProto, protocmp.Transform()))
}

func valuesExpireAfter(t *testing.T, p *redigo.Pool, key []byte, expireAfter time.Time) {
	hash := getHash(t, p, key)
	require.NotEmpty(t, hash)
	for _, val := range hash {
		var msg ExpiringValue
		err := proto.Unmarshal(val, &msg)
		require.NoError(t, err)
		assert.True(t, msg.ExpiresAt.AsTime().After(expireAfter))
	}
}
