package appCtx

import (
	"context"
	"os"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/infra/cache"
	"wechat-clone/core/shared/infra/discovery"
	"wechat-clone/core/shared/infra/lock"
	"wechat-clone/core/shared/infra/smtp"
	"wechat-clone/core/shared/infra/storage"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/hasher"
	"wechat-clone/core/shared/pkg/pubsub"
	"wechat-clone/core/shared/pkg/webpush"

	es8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Option func(*AppContext)

type AppContext struct {
	cfg           *config.Config
	redisClient   *redis.Client
	db            *gorm.DB
	cache         cache.Cache
	hasher        hasher.Hasher
	paseto        xpaseto.PasetoService
	smtp          smtp.SMTP
	storage       storage.Storage
	consulClient  discovery.ConsulClient
	locker        lock.Lock
	cassandra     *gocql.Session
	elasticsearch *es8.Client
	localBus      *pubsub.Bus
	webPush       webpush.WebPushService
}

func NewAppContext(ctx context.Context, opts ...Option) (*AppContext, error) {
	appCtx := &AppContext{}
	for _, opt := range opts {
		opt(appCtx)
	}
	return appCtx, nil
}

func WithRedisClient(redisClient *redis.Client) Option {
	return func(appCtx *AppContext) {
		appCtx.redisClient = redisClient
	}
}

func WithConfig(cfg *config.Config) Option {
	return func(appCtx *AppContext) {
		appCtx.cfg = cfg
	}
}

func WithCache(cache cache.Cache) Option {
	return func(appCtx *AppContext) {
		appCtx.cache = cache
	}
}

func WithDB(db *gorm.DB) Option {
	return func(appCtx *AppContext) {
		appCtx.db = db
	}
}

func WithHasher(hasher hasher.Hasher) Option {
	return func(appCtx *AppContext) {
		appCtx.hasher = hasher
	}
}

func WithPaseto(paseto xpaseto.PasetoService) Option {
	return func(appCtx *AppContext) {
		appCtx.paseto = paseto
	}
}

func WithSMTP(smtp smtp.SMTP) Option {
	return func(appCtx *AppContext) {
		appCtx.smtp = smtp
	}
}

func WithStorage(storage storage.Storage) Option {
	return func(appCtx *AppContext) {
		appCtx.storage = storage
	}
}

func WithConsulClient(consulClient discovery.ConsulClient) Option {
	return func(appCtx *AppContext) {
		appCtx.consulClient = consulClient
	}
}

func WithLocker(locker lock.Lock) Option {
	return func(appCtx *AppContext) {
		appCtx.locker = locker
	}
}

func WithCassandraSession(session *gocql.Session) Option {
	return func(appCtx *AppContext) {
		appCtx.cassandra = session
	}
}

func WithElasticsearchClient(client *es8.Client) Option {
	return func(appCtx *AppContext) {
		appCtx.elasticsearch = client
	}
}

func WithLocalBus(bus *pubsub.Bus) Option {
	return func(appCtx *AppContext) {
		appCtx.localBus = bus
	}
}

func WithWebPush(service webpush.WebPushService) Option {
	return func(appCtx *AppContext) {
		appCtx.webPush = service
	}
}

func (appCtx *AppContext) GetRedisClient() *redis.Client {
	return appCtx.redisClient
}

func (appCtx *AppContext) GetConfig() *config.Config {
	return appCtx.cfg
}

func (appCtx *AppContext) GetDB() *gorm.DB {
	if os.Getenv("ENVIRONMENT") != "production" {
		return appCtx.db.Debug()
	}
	return appCtx.db
}

func (appCtx *AppContext) GetCache() cache.Cache {
	return appCtx.cache
}

func (appCtx *AppContext) GetHasher() hasher.Hasher {
	return appCtx.hasher
}

func (appCtx *AppContext) GetPaseto() xpaseto.PasetoService {
	return appCtx.paseto
}

func (appCtx *AppContext) GetSMTP() smtp.SMTP {
	return appCtx.smtp
}

func (appCtx *AppContext) GetStorage() storage.Storage {
	return appCtx.storage
}

func (appCtx *AppContext) GetConsulClient() discovery.ConsulClient {
	return appCtx.consulClient
}

func (appCtx *AppContext) Locker() lock.Lock {
	return appCtx.locker
}

func (appCtx *AppContext) GetCassandraSession() *gocql.Session {
	return appCtx.cassandra
}

func (appCtx *AppContext) GetElasticsearchClient() *es8.Client {
	return appCtx.elasticsearch
}

func (appCtx *AppContext) LocalBus() *pubsub.Bus {
	return appCtx.localBus
}

func (appCtx *AppContext) GetWebPush() webpush.WebPushService {
	return appCtx.webPush
}

func (appCtx *AppContext) Close() {
	if appCtx.localBus != nil {
		appCtx.localBus.Close()
	}
	if appCtx.cassandra != nil {
		appCtx.cassandra.Close()
	}
	if appCtx.redisClient != nil {
		appCtx.redisClient.Close()
	}
	if appCtx.db != nil {
		ins, _ := appCtx.db.DB()
		ins.Close()
	}
}
