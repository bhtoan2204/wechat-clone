package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/pkg/stackErr"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var DEFAULT_CACHE_EXPIRATION_TIME = time.Hour * 24 * 7

var _ Cache = (*cache)(nil)

//go:generate mockgen -package=cache -destination=cache_mock.go -source=cache.go
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	SetObject(ctx context.Context, key string, val interface{}, duration time.Duration) error
	Delete(ctx context.Context, key string) error
	SetWithDuration(ctx context.Context, key string, value []byte, duration time.Duration) error
	SetExpireTime(ctx context.Context, key string, seconds int64) error
	Exists(ctx context.Context, keys ...string) int64

	LSet(ctx context.Context, key string, vals []byte) error
	LLen(ctx context.Context, key string) (int64, error)
	LGet(ctx context.Context, key string) ([]byte, error)
	LList(ctx context.Context, key string) ([]string, error)
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	DecrBy(ctx context.Context, key string, value int64) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	SetVal(ctx context.Context, key string, value string) error
	SetValWithExp(ctx context.Context, key string, value string, seconds int64) error
	GetVal(ctx context.Context, key string) (string, error)
	LRange(ctx context.Context, key string, from int, to int) ([]string, error)
	ZAdd(ctx context.Context, key string, score float64, member string) error
	ZRange(ctx context.Context, key string, start int64, stop int64) ([]string, error)
	ZRemRangeByRank(ctx context.Context, key string, start int64, stop int64) error
	ZRemRangeByScore(ctx context.Context, key string, min float64, max float64) error
	ZCard(ctx context.Context, key string) (int64, error)
	ZIncrBy(ctx context.Context, key string, increment float64, member string) error
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error)
	ZRevRank(ctx context.Context, key string, member string) (int64, error)
	ZScore(ctx context.Context, key string, member string) (float64, error)

	GetSMembers(ctx context.Context, key string) ([]string, error)
	SetSAdd(ctx context.Context, key string, members ...interface{}) error
	SetNX(ctx context.Context, key string, seconds int64, data interface{}) (bool, error)
	Select(ctx context.Context, index int) error
}

type cache struct {
	rc        *redis.Client
	cacheTime time.Duration
}

func New(rc *redis.Client, cacheTime time.Duration) Cache {
	return &cache{
		rc:        rc,
		cacheTime: cacheTime,
	}
}

func (c *cache) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.rc.Get(ctx, key).Bytes()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get key=%s failed err=%w", key, err))
	}
	return value, nil
}

func (c *cache) Set(ctx context.Context, key string, value []byte) error {
	err := c.rc.Set(ctx, key, value, c.cacheTime).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("set value=%v with key=%s to redis failed err=%w", value, key, err))
	}
	return nil
}

func (c *cache) SetObject(ctx context.Context, key string, val interface{}, duration time.Duration) error {
	dataBytes, err := json.Marshal(val)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := c.rc.Set(ctx, key, dataBytes, duration).Err(); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (c *cache) Delete(ctx context.Context, key string) error {
	err := c.rc.Del(ctx, key).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("delete with key=%s to redis failed err=%w", key, err))
	}
	return nil
}

func (c *cache) SetWithDuration(ctx context.Context, key string, value []byte, duration time.Duration) error {
	err := c.rc.Set(ctx, key, value, duration).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("set value=%v with key=%s and duration=%d to redis failed err=%w", value, key, duration, err))
	}
	return nil
}

func (c *cache) LGet(ctx context.Context, key string) ([]byte, error) {
	value, err := c.rc.LPop(ctx, key).Bytes()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get cache key=%s from redis failed err=%w", key, err))
	}
	return value, nil
}

func (c *cache) LSet(ctx context.Context, key string, val []byte) error {
	if err := c.rc.LPush(ctx, key, val).Err(); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (c *cache) LLen(ctx context.Context, key string) (int64, error) {
	val, err := c.rc.LLen(ctx, key).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("get len of key=%s from redis failed err=%w", key, err))
	}
	return val, nil
}

func (c *cache) LList(ctx context.Context, key string) ([]string, error) {
	vals, err := c.rc.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get datas of key=%s from redis failed err=%w", key, err))
	}
	return vals, nil
}

func (c *cache) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.rc.Decr(ctx, key).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("decrby key=%s from redis failed err=%w", key, err))
	}
	return val, nil
}

func (c *cache) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.rc.Incr(ctx, key).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("incrBy key=%s from redis failed err=%w", key, err))
	}
	return val, nil
}

func (c *cache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.rc.DecrBy(ctx, key, value).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("decrby key=%s from redis failed err=%w", key, err))
	}
	return val, nil
}

func (c *cache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.rc.IncrBy(ctx, key, value).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("incrBy key=%s from redis failed err=%w", key, err))
	}
	return val, nil
}

func (c *cache) LRange(ctx context.Context, key string, from int, to int) ([]string, error) {
	result, err := c.rc.LRange(ctx, key, int64(from), int64(to)).Result()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("lrange cache key=%s from redis failed err=%w", key, err))
	}
	return result, nil
}

func (c *cache) ZIncrBy(ctx context.Context, key string, increment float64, member string) error {
	err := c.rc.
		ZIncrBy(ctx, key, increment, member).
		Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("zincrby member=%v with key=%s and increment=%v to redis failed. Error: %w", member, key, increment, err))
	}

	return nil
}

func (c *cache) ZAdd(ctx context.Context, key string, score float64, member string) error {
	err := c.rc.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("zadd member=%v with key=%s and score=%v to redis failed, err=%w", member, key, score, err))
	}

	return nil
}

func (c *cache) ZRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	result, err := c.rc.ZRange(ctx, key, start, stop).Result()

	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("zrange with key=%s, start=%v and stop=%v to redis failed, err=%w", key, start, stop, err))
	}

	return result, nil
}

func (c *cache) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	result, err := c.rc.ZRevRangeWithScores(ctx, key, start, stop).Result()

	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("ZRevRangeWithScores with key=%s, start=%v and stop=%v to redis failed. Error: %w", key, start, stop, err))
	}

	return result, nil
}

func (c *cache) ZRevRank(ctx context.Context, key string, member string) (int64, error) {
	result, err := c.rc.ZRevRank(ctx, key, member).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("ZRevRank with key=%s, member=%v. Error: %w", key, member, err))
	}

	return result, nil
}

func (c *cache) ZScore(ctx context.Context, key string, member string) (float64, error) {
	result, err := c.rc.ZScore(ctx, key, member).Result()
	if errors.Is(err, redis.Nil) {
		return -1, stackErr.Error(err)
	}

	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("ZRevRank with key=%s, member=%v. Error: %w", key, member, err))
	}

	return result, nil
}

func (c *cache) ZRemRangeByRank(ctx context.Context, key string, start int64, stop int64) error {
	err := c.rc.ZRemRangeByRank(ctx, key, start, stop).Err()

	if err != nil {
		return stackErr.Error(fmt.Errorf("zremrangebyrank with key=%s, start=%v and stop=%v to redis failed, err=%w", key, start, stop, err))
	}

	return nil
}

func (c *cache) ZRemRangeByScore(ctx context.Context, key string, min float64, max float64) error {
	minStr := strconv.FormatFloat(min, 'f', -1, 64)
	maxStr := strconv.FormatFloat(max, 'f', -1, 64)
	err := c.rc.ZRemRangeByScore(ctx, key, minStr, maxStr).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("zremrangebyscore with key=%s, min=%v and max=%v to redis failed, err=%w", key, min, max, err))
	}
	return nil
}

func (c *cache) ZCard(ctx context.Context, key string) (int64, error) {
	val, err := c.rc.ZCard(ctx, key).Result()
	if err != nil {
		return -1, stackErr.Error(fmt.Errorf("zcard with key=%s to redis failed, err=%w", key, err))
	}
	return val, nil
}

func (c *cache) SetVal(ctx context.Context, key string, value string) error {
	err := c.rc.Set(ctx, key, value, c.cacheTime).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("set value=%v with key=%s to redis failed, err=%w", value, key, err))
	}
	return nil
}

func (c *cache) SetValWithExp(ctx context.Context, key string, value string, seconds int64) error {
	err := c.rc.Set(ctx, key, value, time.Duration(seconds)*time.Second).Err()
	if err != nil {
		return stackErr.Error(fmt.Errorf("set value=%v with key=%s to redis failed, err=%w", value, key, err))
	}
	return nil
}

func (c *cache) GetVal(ctx context.Context, key string) (string, error) {
	val, err := c.rc.Get(ctx, key).Result()
	if err != nil {
		return "", stackErr.Error(err)
	}
	return val, nil
}

func (c *cache) SetExpireTime(ctx context.Context, key string, seconds int64) error {
	if err := c.rc.Expire(ctx, key, time.Second*time.Duration(seconds)).Err(); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (c *cache) Exists(ctx context.Context, keys ...string) int64 {
	return c.rc.Exists(ctx, keys...).Val()
}

func (c *cache) GetSMembers(ctx context.Context, key string) ([]string, error) {
	data, err := c.rc.SMembers(ctx, key).Result()
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get smembers with key=%s to redis failed, err=%w", key, err))
	}
	return data, nil
}

func (c *cache) SetSAdd(ctx context.Context, key string, members ...interface{}) error {
	if err := c.rc.SAdd(ctx, key, members).Err(); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (c *cache) SetNX(ctx context.Context, key string, seconds int64, data interface{}) (bool, error) {
	ok, err := c.rc.SetNX(ctx, key, data, time.Second*time.Duration(seconds)).Result()
	if err != nil {
		return false, stackErr.Error(err)
	}
	return ok, nil
}

func (c *cache) Select(ctx context.Context, index int) error {
	if err := c.rc.Do(ctx, "SELECT", index).Err(); err != nil {
		return stackErr.Error(err)
	}
	return nil
}
