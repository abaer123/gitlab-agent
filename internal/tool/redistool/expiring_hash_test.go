package redistool

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
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	redisURLEnvName = "REDIS_URL"
	ttl             = time.Second
)

func TestExpiringHash_Set(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))

	equalHash(t, client, key, 123, value)
}

func TestExpiringHash_Unset(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))
	require.NoError(t, hash.Unset(context.Background(), key, 123))

	require.Empty(t, getHash(t, client, key))
}

func TestExpiringHash_Expires(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))
	time.Sleep(ttl + 100*time.Millisecond)

	require.Empty(t, getHash(t, client, key))
}

func TestExpiringHash_GC(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))
	_, err := client.Pipelined(context.Background(), func(p redis.Pipeliner) error {
		newExpireIn := 3 * ttl
		p.PExpire(context.Background(), key, newExpireIn)
		return nil
	})
	require.NoError(t, err)
	time.Sleep(ttl + 100*time.Millisecond)
	require.NoError(t, hash.Set(context.Background(), key, 321, value))

	keysDeleted, err := hash.GC(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, keysDeleted)

	equalHash(t, client, key, 321, value)
}

func TestExpiringHash_Refresh(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))
	registrationTime := time.Now()
	time.Sleep(ttl / 2)
	require.NoError(t, hash.Refresh(context.Background()))

	// Then
	expireAfter := registrationTime.Add(ttl)
	valuesExpireAfter(t, client, key, expireAfter)
}

func TestExpiringHash_ScanEmpty(t *testing.T) {
	_, hash, key, _ := setupHash(t)

	keysDeleted, err := hash.Scan(context.Background(), key, func(value *anypb.Any, err error) (bool, error) {
		require.NoError(t, err)
		assert.FailNow(t, "unexpected callback invocation")
		return false, nil
	})
	require.NoError(t, err)
	assert.Zero(t, keysDeleted)
}

func TestExpiringHash_Scan(t *testing.T) {
	_, hash, key, value := setupHash(t)

	keysDeleted, err := hash.Scan(context.Background(), key, func(v *anypb.Any, err error) (bool, error) {
		require.NoError(t, err)
		assert.Empty(t, cmp.Diff(value, v, protocmp.Transform()))
		return false, nil
	})
	require.NoError(t, err)
	assert.Zero(t, keysDeleted)
}

func TestExpiringHash_ScanGC(t *testing.T) {
	client, hash, key, value := setupHash(t)

	require.NoError(t, hash.Set(context.Background(), key, 123, value))
	_, err := client.Pipelined(context.Background(), func(p redis.Pipeliner) error {
		newExpireIn := 3 * ttl
		p.PExpire(context.Background(), key, newExpireIn)
		return nil
	})
	require.NoError(t, err)
	time.Sleep(ttl + 100*time.Millisecond)
	require.NoError(t, hash.Set(context.Background(), key, 321, value))

	keysDeleted, err := hash.Scan(context.Background(), key, func(v *anypb.Any, err error) (bool, error) {
		require.NoError(t, err)
		assert.Empty(t, cmp.Diff(value, v, protocmp.Transform()))
		return false, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, keysDeleted)
}

func setupHash(t *testing.T) (redis.UniversalClient, *ExpiringHash, string, *anypb.Any) {
	t.Parallel()
	client := redisClient(t)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := make([]byte, 32)
	_, err := r.Read(prefix)
	require.NoError(t, err)
	key := string(prefix)
	hash := NewExpiringHash(zaptest.NewLogger(t), client, func(key interface{}) string {
		return key.(string)
	}, ttl)
	return client, hash, key, &anypb.Any{
		TypeUrl: "bla",
		Value:   []byte{1, 2, 3},
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

func equalHash(t *testing.T, client redis.UniversalClient, key string, hashKey int64, value *anypb.Any) {
	hash := getHash(t, client, key)
	require.Len(t, hash, 1)
	connectionIdStr := strconv.FormatInt(hashKey, 10)
	require.Contains(t, hash, connectionIdStr)
	val := hash[connectionIdStr]
	var msg ExpiringValue
	err := proto.Unmarshal([]byte(val), &msg)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(value, msg.Value, protocmp.Transform()))
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
