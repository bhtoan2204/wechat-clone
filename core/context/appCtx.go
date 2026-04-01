package appCtx

import (
	"context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/infra/cache"
	"go-socket/core/shared/infra/discovery"
	"go-socket/core/shared/infra/smtp"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Option func(*AppContext)

type AppContext struct {
	cfg          *config.Config
	redisClient  *redis.Client
	db           *gorm.DB
	cache        cache.Cache
	hasher       hasher.Hasher
	paseto       xpaseto.PasetoService
	smtp         smtp.SMTP
	consulClient discovery.ConsulClient
	services     map[string]interface{}
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

func WithConsulClient(consulClient discovery.ConsulClient) Option {
	return func(appCtx *AppContext) {
		appCtx.consulClient = consulClient
	}
}

func (appCtx *AppContext) GetRedisClient() *redis.Client {
	return appCtx.redisClient
}

func (appCtx *AppContext) GetConfig() *config.Config {
	return appCtx.cfg
}

func (appCtx *AppContext) GetDB() *gorm.DB {
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

func (appCtx *AppContext) GetConsulClient() discovery.ConsulClient {
	return appCtx.consulClient
}

func (appCtx *AppContext) Close() {
	appCtx.redisClient.Close()
	if appCtx.db != nil {
		ins, _ := appCtx.db.DB()
		ins.Close()
	}
}

func (appCtx *AppContext) RegisterService(name string, service interface{}) {
	if appCtx.services == nil {
		appCtx.services = make(map[string]interface{})
	}
	appCtx.services[name] = service
}

func (appCtx *AppContext) GetService(name string) (interface{}, bool) {
	if appCtx.services == nil {
		return nil, false
	}
	service, ok := appCtx.services[name]
	return service, ok
}
