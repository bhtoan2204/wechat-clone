package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-socket/core/modules/account/domain/entity"
	sharedcache "go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/redis/go-redis/v9"
)

//go:generate mockgen -package=cache -destination=session_cache_mock.go -source=session_cache.go
type SessionCache interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, bool, error)
	Set(ctx context.Context, session *entity.Session) error
	Delete(ctx context.Context, sessionID string) error
}

type sessionCache struct {
	cache sharedcache.Cache
}

func NewSessionCache(cache sharedcache.Cache) SessionCache {
	return &sessionCache{cache: cache}
}

func sessionCacheKey(sessionID string) string {
	return "account:session:" + sessionID
}

func (c *sessionCache) Get(ctx context.Context, sessionID string) (*entity.Session, bool, error) {
	if c == nil || c.cache == nil {
		return nil, false, nil
	}
	data, err := c.cache.Get(ctx, sessionCacheKey(sessionID))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, stackErr.Error(err)
	}

	var session entity.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, false, stackErr.Error(fmt.Errorf("unmarshal session cache failed: %v", err))
	}
	return &session, true, nil
}

func (c *sessionCache) Set(ctx context.Context, session *entity.Session) error {
	if c == nil || c.cache == nil || session == nil {
		return nil
	}
	data, err := json.Marshal(session)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal session cache failed: %v", err))
	}
	return c.cache.Set(ctx, sessionCacheKey(session.ID), data)
}

func (c *sessionCache) Delete(ctx context.Context, sessionID string) error {
	if c == nil || c.cache == nil {
		return nil
	}
	return c.cache.Delete(ctx, sessionCacheKey(sessionID))
}
