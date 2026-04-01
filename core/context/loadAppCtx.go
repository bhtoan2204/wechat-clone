package appCtx

import (
	"context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/constant"
	"go-socket/core/shared/infra/cache"
	dbinfra "go-socket/core/shared/infra/db"
	"go-socket/core/shared/infra/discovery"
	"go-socket/core/shared/infra/redis"
	"go-socket/core/shared/infra/smtp"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"time"
)

func LoadAppCtx(ctx context.Context, cfg *config.Config) (*AppContext, error) {
	var opts []Option
	opts = append(opts, WithConfig(cfg))

	db, err := dbinfra.NewConnection(ctx, cfg)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	opts = append(opts, WithDB(db))

	redisClient, err := redis.NewStandaloneRedisClient(cfg)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	opts = append(opts, WithRedisClient(redisClient))

	cache := cache.New(redisClient, constant.DEFAULT_CACHE_EXPIRATION_TIME)
	opts = append(opts, WithCache(cache))

	hasher, err := hasher.NewHasher()
	if err != nil {
		return nil, stackerr.Error(err)
	}
	opts = append(opts, WithHasher(hasher))

	paseto, err := xpaseto.NewPaseto(cfg.AuthConfig.PasetoKey, cfg.AuthConfig.TokenIssuer, time.Duration(cfg.AuthConfig.AccessTokenTTLSeconds)*time.Second)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	opts = append(opts, WithPaseto(paseto))

	smtpClient := smtp.NewSMTP()
	opts = append(opts, WithSMTP(smtpClient))

	consulClient, err := discovery.NewConsulClient(ctx, cfg)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	opts = append(opts, WithConsulClient(consulClient))

	return NewAppContext(ctx, opts...)
}
