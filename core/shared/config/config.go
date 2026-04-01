package config

type Config struct {
	ServerConfig   ServerConfig
	RedisConfig    RedisConfig
	DBConfig       DBConfig
	AuthConfig     AuthConfig
	KafkaConfig    KafkaConfig
	SecurityConfig SecurityConfig
	WebPushConfig  WebPushConfig
	ConsulConfig   ConsulConfig
	LedgerConfig   LedgerConfig
}

type ServerConfig struct {
	Environment string `env:"ENVIRONMENT"`
	Port        int    `env:"SERVER_PORT,default=0"`
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

type DBConfig struct {
	ConnectionURL          string `env:"DB_CONNECTION_URL"`
	Driver                 string `env:"DB_DRIVER"`
	MaxOpenConnNumber      int    `env:"DB_MAX_OPEN_CONN_NUMBER"`
	MaxIdleConnNumber      int    `env:"DB_MAX_IDLE_CONN_NUMBER"`
	ConnMaxLifeTimeSeconds int64  `env:"DB_CONN_MAX_LIFE_TIME_SECONDS"`
}

type AuthConfig struct {
	PasetoKey             string `env:"AUTH_PASETO_KEY" json:"-"`
	TokenIssuer           string `env:"AUTH_TOKEN_ISSUER"`
	AccessTokenTTLSeconds int64  `env:"AUTH_ACCESS_TOKEN_TTL_SECONDS"`
}

type KafkaConfig struct {
	KafkaServers              string `env:"KAFKA_SERVERS"`
	KafkaOffsetReset          string `env:"KAFKA_OFFSET_RESET"`
	KafkaNotificationConsumer KafkaNotificationConsumer
	KafkaPaymentConsumer      KafkaPaymentConsumer
}

type KafkaNotificationConsumer struct {
	NotificationGroup string `env:"KAFKA_NOTIFICATION_CONSUMER_GROUP"`

	AccountTopic string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
}

type KafkaPaymentConsumer struct {
	PaymentGroup string `env:"KAFKA_PAYMENT_CONSUMER_GROUP"`

	AccountTopic       string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
	PaymentEventsTopic string `env:"KAFKA_CONSUMER_PAYMENT_EVENTS_TOPIC"`
}

type SecurityConfig struct {
	SecretKey string `env:"SECURITY_SECRET_KEY" json:"-"`
}

type WebPushConfig struct {
	VAPIDPublicKey  string `env:"WEBPUSH_VAPID_PUBLIC_KEY"`
	VAPIDPrivateKey string `env:"WEBPUSH_VAPID_PRIVATE_KEY" json:"-"`
	TTL             int    `env:"WEBPUSH_TTL"`
}

type ConsulConfig struct {
	Address    string `env:"CONSUL_ADDRESS"`
	Scheme     string `env:"CONSUL_SCHEME"`
	DataCenter string `env:"CONSUL_DATA_CENTER"`
	Token      string `env:"CONSUL_TOKEN"`
}

type LedgerConfig struct {
	MockWebhookSecret string `env:"LEDGER_MOCK_WEBHOOK_SECRET,default=mock-secret"`
	Stripe            LedgerStripeConfig
}

type LedgerStripeConfig struct {
	SecretKey     string `env:"LEDGER_STRIPE_SECRET_KEY"`
	WebhookSecret string `env:"LEDGER_STRIPE_WEBHOOK_SECRET"`
	SuccessURL    string `env:"LEDGER_STRIPE_SUCCESS_URL"`
	CancelURL     string `env:"LEDGER_STRIPE_CANCEL_URL"`
}
