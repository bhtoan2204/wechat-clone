package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/redis/go-redis/v9"
)

type AccountCache interface {
	Get(ctx context.Context, id string) (*entity.Account, bool, error)
	Set(ctx context.Context, m *entity.Account) error
	Delete(ctx context.Context, id string) error
	GetByEmail(ctx context.Context, email string) (*entity.Account, bool, error)
	SetByEmail(ctx context.Context, m *entity.Account) error
	DeleteByEmail(ctx context.Context, email string) error
}

type accountCache struct {
	cache cache.Cache
}

func NewAccountCache(cache cache.Cache) AccountCache {
	return &accountCache{cache: cache}
}

func accountCacheKey(id string) string {
	return "account:" + id
}

func accountEmailCacheKey(email string) string {
	return "account:email:" + email
}

func (a *accountCache) Get(ctx context.Context, id string) (*entity.Account, bool, error) {
	if a == nil || a.cache == nil {
		return nil, false, nil
	}
	data, err := a.cache.Get(ctx, accountCacheKey(id))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, stackErr.Error(err)
	}
	var m entity.Account
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, stackErr.Error(fmt.Errorf("unmarshal account cache failed: %v", err))
	}
	return &m, true, nil
}

func (a *accountCache) Set(ctx context.Context, m *entity.Account) error {
	if a == nil || a.cache == nil || m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal account cache failed: %v", err))
	}
	return a.cache.Set(ctx, accountCacheKey(m.ID), data)
}

func (a *accountCache) Delete(ctx context.Context, id string) error {
	if a == nil || a.cache == nil {
		return nil
	}
	return a.cache.Delete(ctx, accountCacheKey(id))
}

func (a *accountCache) GetByEmail(ctx context.Context, email string) (*entity.Account, bool, error) {
	if a == nil || a.cache == nil {
		return nil, false, nil
	}
	data, err := a.cache.Get(ctx, accountEmailCacheKey(email))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, stackErr.Error(err)
	}
	var m entity.Account
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, stackErr.Error(fmt.Errorf("unmarshal account cache failed: %v", err))
	}
	return &m, true, nil
}

func (a *accountCache) SetByEmail(ctx context.Context, m *entity.Account) error {
	if a == nil || a.cache == nil || m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal account cache failed: %v", err))
	}
	return a.cache.Set(ctx, accountEmailCacheKey(m.Email.Value()), data)
}

func (a *accountCache) DeleteByEmail(ctx context.Context, email string) error {
	if a == nil || a.cache == nil {
		return nil
	}
	return a.cache.Delete(ctx, accountEmailCacheKey(email))
}
