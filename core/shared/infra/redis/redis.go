package redis

import (
	"fmt"
	"go-socket/core/shared/config"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewStandaloneRedisClient(cfg *config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisConfig.ConnectionURL)
	if err != nil {
		return nil, stackerr.Error(fmt.Errorf("parse url failed err=%w", err))
	}
	opts.PoolSize = cfg.RedisConfig.PoolSize
	opts.DialTimeout = time.Duration(cfg.RedisConfig.DialTimeoutSeconds) * time.Second
	opts.ReadTimeout = time.Duration(cfg.RedisConfig.ReadTimeoutSeconds) * time.Second
	opts.WriteTimeout = time.Duration(cfg.RedisConfig.WriteTimeoutSeconds) * time.Second
	opts.MaxIdleConns = cfg.RedisConfig.MaxIdleConnNumber
	opts.MaxActiveConns = cfg.RedisConfig.MaxActiveConnNumber

	return redis.NewClient(opts), nil
}
