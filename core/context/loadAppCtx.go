package appCtx

import (
	"context"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/constant"
	"wechat-clone/core/shared/infra/cache"
	cassandraclient "wechat-clone/core/shared/infra/cassandra"
	dbinfra "wechat-clone/core/shared/infra/db"
	"wechat-clone/core/shared/infra/discovery"
	elasticclient "wechat-clone/core/shared/infra/elasticsearch"
	"wechat-clone/core/shared/infra/lock"
	"wechat-clone/core/shared/infra/redis"
	"wechat-clone/core/shared/infra/smtp"
	"wechat-clone/core/shared/infra/storage"
	"wechat-clone/core/shared/infra/temporalclient"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/hasher"
	"wechat-clone/core/shared/pkg/pubsub"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/pkg/webpush"
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

	smtpClient := smtp.NewSMTP(cfg)
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

	localBus := pubsub.New(pubsub.Config{
		BufferSize:  256,
		PublishMode: pubsub.PublishBlocking,
	})
	opts = append(opts, WithLocalBus(localBus))

	webPushService, err := webpush.NewWebPush(cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithWebPush(webPushService))

	temporalClient, err := temporalclient.NewTemporalClient(ctx, cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	opts = append(opts, WithTemporalClient(temporalClient))

	return NewAppContext(ctx, opts...)
}
