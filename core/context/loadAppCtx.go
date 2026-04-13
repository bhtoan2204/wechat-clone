package appCtx

import (
	"context"
	"go-socket/core/modules/account/infra/lock"
	"go-socket/core/shared/config"
	"go-socket/core/shared/constant"
	"go-socket/core/shared/infra/cache"
	cassandraclient "go-socket/core/shared/infra/cassandra"
	dbinfra "go-socket/core/shared/infra/db"
	"go-socket/core/shared/infra/discovery"
	elasticclient "go-socket/core/shared/infra/elasticsearch"
	"go-socket/core/shared/infra/redis"
	"go-socket/core/shared/infra/smtp"
	"go-socket/core/shared/infra/storage"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/stackErr"
)

func LoadAppCtx(ctx context.Context, cfg *config.Config) (*AppContext, error) {
	var opts []Option
	opts = append(opts, WithConfig(cfg))

	db, err := dbinfra.NewConnection(ctx, cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithDB(db))

	redisClient, err := redis.NewStandaloneRedisClient(cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithRedisClient(redisClient))

	cache := cache.New(redisClient, constant.DEFAULT_CACHE_EXPIRATION_TIME)
	opts = append(opts, WithCache(cache))

	hasher, err := hasher.NewHasher()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithHasher(hasher))

	paseto, err := xpaseto.NewPaseto(cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithPaseto(paseto))

	smtpClient := smtp.NewSMTP()
	opts = append(opts, WithSMTP(smtpClient))

	objectStorage, err := storage.NewMinIO(cfg.StorageConfig)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithStorage(objectStorage))

	consulClient, err := discovery.NewConsulClient(ctx, cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithConsulClient(consulClient))

	locker := lock.NewLock(redisClient)
	opts = append(opts, WithLocker(locker))

	cassandraSession, err := cassandraclient.NewSession(ctx, cfg.CassandraConfig)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithCassandraSession(cassandraSession))

	elasticsearchClient, err := elasticclient.NewClient(cfg.ElasticsearchConfig)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithElasticsearchClient(elasticsearchClient))

	return NewAppContext(ctx, opts...)
}
