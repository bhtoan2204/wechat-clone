package idempotency

import (
	"context"
	"gateway/pkg/cache"
	"time"
)

const keyPrefix = "idempotency:"

const (
	DEFAULT_IDEMPOTENCY_LOCK_TTL = time.Minute * 5
	DEFAULT_IDEMPOTENCY_DONE_TTL = time.Hour * 24
)

type RedisStore struct {
	cache cache.Cache
}

func NewRedisStore(cache cache.Cache) *RedisStore {
	return &RedisStore{cache: cache}
}

func (s *RedisStore) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s == nil || s.cache == nil {
		return true, nil
	}
	seconds := int64(ttl.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return s.cache.SetNX(ctx, keyPrefix+key, seconds, "locked")
}

func (s *RedisStore) MarkDone(ctx context.Context, key string, ttl time.Duration) error {
	if s == nil || s.cache == nil {
		return nil
	}
	seconds := int64(ttl.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return s.cache.SetValWithExp(ctx, keyPrefix+key, "done", seconds)
}

func (s *RedisStore) Release(ctx context.Context, key string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Delete(ctx, keyPrefix+key)
}
