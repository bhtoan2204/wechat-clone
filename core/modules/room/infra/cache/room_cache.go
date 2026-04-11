package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/redis/go-redis/v9"
)

type RoomCache struct {
	cache cache.Cache
}

func NewRoomCache(cache cache.Cache) *RoomCache {
	return &RoomCache{cache: cache}
}

func (r *RoomCache) Get(ctx context.Context, id string) (*entity.Room, bool, error) {
	if r == nil || r.cache == nil {
		return nil, false, nil
	}
	data, err := r.cache.Get(ctx, roomCacheKey(id))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, stackErr.Error(err)
	}
	var m entity.Room
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, stackErr.Error(fmt.Errorf("unmarshal room cache failed: %v", err))
	}
	return &m, true, nil
}

func (r *RoomCache) Set(ctx context.Context, m *entity.Room) error {
	if r == nil || r.cache == nil || m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal room cache failed: %v", err))
	}
	return stackErr.Error(r.cache.Set(ctx, roomCacheKey(m.ID), data))
}

func (r *RoomCache) Delete(ctx context.Context, id string) error {
	if r == nil || r.cache == nil {
		return nil
	}
	return stackErr.Error(r.cache.Delete(ctx, roomCacheKey(id)))
}

func roomCacheKey(id string) string {
	return "room:" + id
}
