package redistool

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// KeyToRedisKey is used to convert key1 in
// HSET key1 key2 value.
type KeyToRedisKey func(key interface{}) string
type ScanCallback func(value *anypb.Any, err error) (bool /* done */, error)

type ExpiringHashInterface interface {
	Set(ctx context.Context, key interface{}, hashKey int64, value *anypb.Any) error
	Unset(ctx context.Context, key interface{}, hashKey int64) error
	Scan(ctx context.Context, key interface{}, cb ScanCallback) (int /* keysDeleted */, error)
	// GC iterates all relevant stored data and deletes expired entries.
	// It returns number of deleted Redis (hash) keys, including when an error occurs.
	// It only inspects/GCs hashes where it has entries. Other concurrent clients GC same and/or other corresponding hashes.
	// Hashes that don't have a corresponding client (e.g. because it crashed) will expire because of TTL on the hash key.
	GC(ctx context.Context) (int /* keysDeleted */, error)
	Refresh(ctx context.Context) error
}

type ExpiringHash struct {
	log           *zap.Logger
	client        redis.UniversalClient
	keyToRedisKey KeyToRedisKey
	ttl           time.Duration
	data          map[interface{}]map[int64]*anypb.Any // key -> hash key -> value
}

func NewExpiringHash(log *zap.Logger, client redis.UniversalClient, keyToRedisKey KeyToRedisKey, ttl time.Duration) *ExpiringHash {
	return &ExpiringHash{
		log:           log,
		client:        client,
		keyToRedisKey: keyToRedisKey,
		ttl:           ttl,
		data:          make(map[interface{}]map[int64]*anypb.Any),
	}
}

func (h *ExpiringHash) Set(ctx context.Context, key interface{}, hashKey int64, value *anypb.Any) error {
	h.setData(key, hashKey, value)
	return h.refreshKey(ctx, key, map[int64]*anypb.Any{
		hashKey: value,
	})
}

func (h *ExpiringHash) Unset(ctx context.Context, key interface{}, hashKey int64) error {
	h.unsetData(key, hashKey)
	return h.client.HDel(ctx, h.keyToRedisKey(key), strconv.FormatInt(hashKey, 10)).Err()
}

func (h *ExpiringHash) Scan(ctx context.Context, key interface{}, cb ScanCallback) (keysDeleted int, retErr error) {
	redisKey := h.keyToRedisKey(key)
	var keysToDelete []string
	defer func() {
		if len(keysToDelete) == 0 {
			return
		}
		_, err := h.client.HDel(ctx, redisKey, keysToDelete...).Result()
		if err != nil {
			if retErr == nil {
				retErr = err
			}
			return
		}
		keysDeleted = len(keysToDelete)
	}()
	// Scan keys of a hash. See https://redis.io/commands/scan
	iter := h.client.HScan(ctx, redisKey, 0, "", 0).Iterator()
	for iter.Next(ctx) {
		k := iter.Val()
		if !iter.Next(ctx) {
			err := iter.Err()
			if err != nil {
				return 0, err
			}
			// This shouldn't happen
			return 0, errors.New("invalid Redis reply")
		}
		v := iter.Val()
		var msg ExpiringValue
		err := proto.Unmarshal([]byte(v), &msg)
		if err != nil {
			var done bool
			done, err = cb(nil, fmt.Errorf("failed to unmarshal hash value from key 0x%x: %w", k, err))
			if err != nil || done {
				return 0, err
			}
			continue // try to skip and continue
		}
		if msg.ExpiresAt != nil && msg.ExpiresAt.AsTime().Before(time.Now()) {
			keysToDelete = append(keysToDelete, k)
			continue
		}
		done, err := cb(msg.Value, nil)
		if err != nil || done {
			return 0, err
		}
	}
	return 0, iter.Err()
}

func (h *ExpiringHash) GC(ctx context.Context) (int, error) {
	var deletedKeys int
	for key := range h.data {
		deleted, err := h.gcHash(ctx, key)
		if err != nil {
			return deletedKeys, err
		}
		deletedKeys += deleted
	}
	return deletedKeys, nil
}

// gcHash iterates a hash and removes all expired values.
// It assumes that values are marshaled ExpiringValue.
func (h *ExpiringHash) gcHash(ctx context.Context, key interface{}) (int, error) {
	return h.Scan(ctx, key, func(value *anypb.Any, err error) (bool, error) {
		// nothing to do
		return false, nil
	})
}

func (h *ExpiringHash) Refresh(ctx context.Context) error {
	for key, hashData := range h.data {
		err := h.refreshKey(ctx, key, hashData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *ExpiringHash) refreshKey(ctx context.Context, key interface{}, hashData map[int64]*anypb.Any) error {
	args := make([]interface{}, 0, len(hashData)*2)
	expiresAt := timestamppb.New(time.Now().Add(h.ttl))
	for hashKey, value := range hashData {
		redisValue, err := proto.Marshal(&ExpiringValue{
			ExpiresAt: expiresAt,
			Value:     value,
		})
		if err != nil {
			// This should never happen
			h.log.Error("Failed to marshal ExpiringValue", zap.Error(err))
			continue
		}
		args = append(args, hashKey, redisValue)
	}
	redisKey := h.keyToRedisKey(key)
	_, err := h.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, redisKey, args)
		p.PExpire(ctx, redisKey, h.ttl)
		return nil
	})
	return err
}

func (h *ExpiringHash) setData(key interface{}, hashKey int64, value *anypb.Any) {
	nm := h.data[key]
	if nm == nil {
		nm = make(map[int64]*anypb.Any, 1)
		h.data[key] = nm
	}
	nm[hashKey] = value
}

func (h *ExpiringHash) unsetData(key interface{}, hashKey int64) {
	nm := h.data[key]
	delete(nm, hashKey)
	if len(nm) == 0 {
		delete(h.data, key)
	}
}
