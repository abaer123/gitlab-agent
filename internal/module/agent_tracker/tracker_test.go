package agent_tracker

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
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
	client, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))

	// Then
	equalHash(t, client, tr.connectionsByProjectIdHashKey(info.ProjectId), info)
	equalHash(t, client, tr.connectionsByAgentIdHashKey(info.Id), info)
}

func TestUnregisterConnection(t *testing.T) {
	client, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	require.NoError(t, tr.unregisterConnection(context.Background(), info))

	// Then
	require.Empty(t, getHash(t, client, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, client, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestHashExpires(t *testing.T) {
	t.Parallel()
	client, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	time.Sleep(tr.ttl + 100*time.Millisecond)

	// Then
	require.Empty(t, getHash(t, client, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, client, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestGC(t *testing.T) {
	t.Parallel()
	client, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	_, err := client.Pipelined(context.Background(), func(p redis.Pipeliner) error {
		newExpireIn := 3 * tr.ttl
		p.PExpire(context.Background(), tr.connectionsByProjectIdHashKey(info.ProjectId), newExpireIn)
		p.PExpire(context.Background(), tr.connectionsByAgentIdHashKey(info.Id), newExpireIn)
		return nil
	})
	require.NoError(t, err)
	time.Sleep(tr.ttl + 100*time.Millisecond)
	deletedKeys, err := tr.runGc(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 2, deletedKeys)

	// Then
	require.Empty(t, getHash(t, client, tr.connectionsByProjectIdHashKey(info.ProjectId)))
	require.Empty(t, getHash(t, client, tr.connectionsByAgentIdHashKey(info.Id)))
}

func TestRefresh(t *testing.T) {
	client, tr := setupTracker(t)

	// Given
	info := connInfo()

	// When
	require.NoError(t, tr.registerConnection(context.Background(), info))
	registrationTime := time.Now()
	oldExpireIn := tr.ttl
	tr.ttl = 2 * tr.ttl // increase
	err := tr.refreshRegistrations(context.Background())
	require.NoError(t, err)

	// Then
	expireAfter := registrationTime.Add(oldExpireIn)
	valuesExpireAfter(t, client, tr.connectionsByProjectIdHashKey(info.ProjectId), expireAfter)
	valuesExpireAfter(t, client, tr.connectionsByAgentIdHashKey(info.Id), expireAfter)
}

func setupTracker(t *testing.T) (redis.UniversalClient, *RedisTracker) {
	client := redisClient(t)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := make([]byte, 32)
	_, err := r.Read(prefix)
	require.NoError(t, err)
	tr := NewRedisTracker(zaptest.NewLogger(t), client, string(prefix), time.Second, time.Minute, time.Minute)
	return client, tr
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

func redisClient(t *testing.T) redis.UniversalClient {
	redisURL := os.Getenv(redisURLEnvName)
	if redisURL == "" {
		t.Skipf("%s environment variable not set, skipping test", redisURLEnvName)
	}

	opts, err := redis.ParseURL(redisURL)
	require.NoError(t, err)
	return redis.NewClient(opts)
}

func getHash(t *testing.T, client redis.UniversalClient, key string) map[string]string {
	reply, err := client.HGetAll(context.Background(), key).Result()
	require.NoError(t, err)
	return reply
}

func equalHash(t *testing.T, client redis.UniversalClient, key string, info *ConnectedAgentInfo) {
	hash := getHash(t, client, key)
	require.Len(t, hash, 1)
	connectionIdStr := strconv.Itoa(int(info.ConnectionId))
	require.Contains(t, hash, connectionIdStr)
	val := hash[connectionIdStr]
	var msg ExpiringValue
	err := proto.Unmarshal([]byte(val), &msg)
	require.NoError(t, err)
	var valProto ConnectedAgentInfo
	err = anypb.UnmarshalTo(msg.Value, &valProto, proto.UnmarshalOptions{})
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(info, &valProto, protocmp.Transform()))
}

func valuesExpireAfter(t *testing.T, client redis.UniversalClient, key string, expireAfter time.Time) {
	hash := getHash(t, client, key)
	require.NotEmpty(t, hash)
	for _, val := range hash {
		var msg ExpiringValue
		err := proto.Unmarshal([]byte(val), &msg)
		require.NoError(t, err)
		assert.True(t, msg.ExpiresAt.AsTime().After(expireAfter))
	}
}
