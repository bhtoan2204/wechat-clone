package config

type Config struct {
	ServerConfig ServerConfig
	RedisConfig  RedisConfig
	HttpConfig   HttpConfig
	DBConfig     DBConfig
	AuthConfig   AuthConfig
	KafkaConfig  KafkaConfig
}

type ServerConfig struct {
	Environment string `env:"ENVIRONMENT"`
}

type RedisConfig struct {
	ConnectionURL       string `env:"REDIS_CONNECTION_URL"`
	PoolSize            int    `env:"REDIS_POOL_SIZE"`
	DialTimeoutSeconds  int    `env:"REDIS_DIAL_TIMEOUT_SECONDS"`
	ReadTimeoutSeconds  int    `env:"REDIS_READ_TIMEOUT_SECONDS"`
	WriteTimeoutSeconds int    `env:"REDIS_WRITE_TIMEOUT_SECONDS"`
	IdleTimeoutSeconds  int    `env:"REDIS_IDLE_TIMEOUT_SECONDS"`
	MaxIdleConnNumber   int    `env:"REDIS_MAX_IDLE_CONN_NUMBER"`
	MaxActiveConnNumber int    `env:"REDIS_MAX_ACTIVE_CONN_NUMBER"`
}

type HttpConfig struct {
	Port int `env:"HTTP_PORT"`
}

type DBConfig struct {
	ConnectionURL          string `env:"DB_CONNECTION_URL"`
	Driver                 string `env:"DB_DRIVER"`
	MaxOpenConnNumber      int    `env:"DB_MAX_OPEN_CONN_NUMBER"`
	MaxIdleConnNumber      int    `env:"DB_MAX_IDLE_CONN_NUMBER"`
	ConnMaxLifeTimeSeconds int64  `env:"DB_CONN_MAX_LIFE_TIME_SECONDS"`
}

type AuthConfig struct {
	PasetoKey             string `env:"AUTH_PASETO_KEY"`
	TokenIssuer           string `env:"AUTH_TOKEN_ISSUER"`
	AccessTokenTTLSeconds int64  `env:"AUTH_ACCESS_TOKEN_TTL_SECONDS"`
}

type KafkaConfig struct {
	KafkaServers              string `env:"KAFKA_SERVERS"`
	KafkaOffsetReset          string `env:"KAFKA_OFFSET_RESET"`
	KafkaNotificationConsumer KafkaNotificationConsumer
}

type KafkaNotificationConsumer struct {
	NotificationGroup string `env:"KAFKA_NOTIFICATION_CONSUMER_GROUP"`

	AccountTopic string `env:"KAFKA_NOTIFICATION_CONSUMER_ACCOUNT_TOPIC"`
}
