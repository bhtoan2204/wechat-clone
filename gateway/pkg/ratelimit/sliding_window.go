package ratelimit

import (
	"context"
	"fmt"
	"gateway/pkg/cache"
	"time"
)

type SlidingWindowLimiter struct {
	cache  cache.Cache
	limit  int64
	window time.Duration
	prefix string
}

func NewSlidingWindowLimiter(cache cache.Cache, limit int64, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		cache:  cache,
		limit:  limit,
		window: window,
		prefix: "ratelimit:",
	}
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if l == nil || l.cache == nil {
		return true, nil
	}
	now := time.Now().UTC()
	windowSeconds := int64(l.window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	member := fmt.Sprintf("%d", now.UnixNano())
	redisKey := l.prefix + key

	if err := l.cache.ZAdd(ctx, redisKey, float64(now.Unix()), member); err != nil {
		return true, nil
	}
	windowStart := float64(now.Unix() - windowSeconds)
	_ = l.cache.ZRemRangeByScore(ctx, redisKey, 0, windowStart)
	count, err := l.cache.ZCard(ctx, redisKey)
	if err != nil {
		return true, nil
	}
	_ = l.cache.SetExpireTime(ctx, redisKey, windowSeconds)
	return count <= l.limit, nil
}
