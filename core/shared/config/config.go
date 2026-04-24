package config

type Config struct {
	ServerConfig        ServerConfig
	RedisConfig         RedisConfig
	DBConfig            DBConfig
	AuthConfig          AuthConfig
	KafkaConfig         KafkaConfig
	SecurityConfig      SecurityConfig
	WebPushConfig       WebPushConfig
	ConsulConfig        ConsulConfig
	LedgerConfig        LedgerConfig
	StorageConfig       StorageConfig
	CassandraConfig     CassandraConfig
	ElasticsearchConfig ElasticsearchConfig
	SMTPConfig          SMTPConfig
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
	AccessPublicKey        string `env:"AUTH_ACCESS_PUBLIC_KEY"`
	AccessPrivateKey       string `env:"AUTH_ACCESS_PRIVATE_KEY"`
	RefreshPublicKey       string `env:"AUTH_REFRESH_PUBLIC_KEY"`
	RefreshPrivateKey      string `env:"AUTH_REFRESH_PRIVATE_KEY"`
	TokenIssuer            string `env:"AUTH_TOKEN_ISSUER"`
	AccessTokenTTLSeconds  int64  `env:"AUTH_ACCESS_TOKEN_TTL_SECONDS"`
	RefreshTokenTTLSeconds int64  `env:"AUTH_REFRESH_TOKEN_TTL_SECONDS"`
	VerifyEmailURL         string `env:"AUTH_VERIFY_EMAIL_URL"`
	GoogleConfig           GoogleConfig
}

type GoogleConfig struct {
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"GOOGLE_CLIENT_REDIRECT_URL"`
}

type KafkaConfig struct {
	KafkaServers              string `env:"KAFKA_SERVERS"`
	KafkaOffsetReset          string `env:"KAFKA_OFFSET_RESET"`
	KafkaNotificationConsumer KafkaNotificationConsumer
	KafkaPaymentConsumer      KafkaPaymentConsumer
	KafkaLedgerConsumer       KafkaLedgerConsumer
	KafkaRoomConsumer         KafkaRoomConsumer
	KafkaRelationshipConsumer KafkaRelationshipConsumer
}

type KafkaNotificationConsumer struct {
	NotificationGroup string `env:"KAFKA_NOTIFICATION_CONSUMER_GROUP"`

	AccountTopic       string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
	RoomOutboxTopic    string `env:"KAFKA_CONSUMER_ROOM_OUTBOX_TOPIC"`
	PaymentOutboxTopic string `env:"KAFKA_CONSUMER_PAYMENT_OUTBOX_TOPIC"`
}

type KafkaPaymentConsumer struct {
	PaymentGroup string `env:"KAFKA_PAYMENT_CONSUMER_GROUP"`

	AccountTopic       string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
	PaymentEventsTopic string `env:"KAFKA_CONSUMER_PAYMENT_EVENTS_TOPIC"`
}

type KafkaRoomConsumer struct {
	RoomMessagingGroup  string `env:"KAFKA_ROOM_CONSUMER_MESSAGING_GROUP"`
	RoomProjectionGroup string `env:"KAFKA_ROOM_CONSUMER_PROJECTION_GROUP"`
	AccountTopic        string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
	RoomOutboxTopic     string `env:"KAFKA_CONSUMER_ROOM_OUTBOX_TOPIC"`
	LedgerOutboxTopic   string `env:"KAFKA_CONSUMER_LEDGER_OUTBOX_TOPIC"`
}

type KafkaLedgerConsumer struct {
	LedgerMessagingGroup  string `env:"KAFKA_LEDGER_CONSUMER_MESSAGING_GROUP"`
	LedgerProjectionGroup string `env:"KAFKA_LEDGER_CONSUMER_PROJECTION_GROUP"`

	PaymentOutboxTopic string `env:"KAFKA_CONSUMER_PAYMENT_OUTBOX_TOPIC"`
	LedgerOutboxTopic  string `env:"KAFKA_CONSUMER_LEDGER_OUTBOX_TOPIC"`
}

type KafkaRelationshipConsumer struct {
	RelationshipProjectionGroup string `env:"KAFKA_RELATIONSHIP_CONSUMER_PROJECTION_GROUP"`
	RelationshipOutboxTopic     string `env:"KAFKA_CONSUMER_RELATIONSHIP_OUTBOX_TOPIC"`
	AccountTopic                string `env:"KAFKA_CONSUMER_ACCOUNT_TOPIC"`
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
	PublicKey                        string `env:"LEDGER_STRIPE_PUBLIC_KEY"`
	SecretKey                        string `env:"LEDGER_STRIPE_SECRET_KEY"`
	WebhookSecret                    string `env:"LEDGER_STRIPE_WEBHOOK_SECRET"`
	SuccessURL                       string `env:"LEDGER_STRIPE_SUCCESS_URL"`
	CancelURL                        string `env:"LEDGER_STRIPE_CANCEL_URL"`
	FeeRateBPS                       int64  `env:"LEDGER_STRIPE_FEE_RATE_BPS,default=0"`
	FeeFlatAmount                    int64  `env:"LEDGER_STRIPE_FEE_FLAT_AMOUNT,default=0"`
	FeeAccountID                     string `env:"LEDGER_STRIPE_FEE_ACCOUNT_ID,default=ledger:fee:provider:stripe"`
	WithdrawalScheduleIntervalSecond int    `env:"LEDGER_STRIPE_WITHDRAWAL_POLL_INTERVAL_SECONDS,default=5"`
	WithdrawalBatchSize              int    `env:"LEDGER_STRIPE_WITHDRAWAL_BATCH_SIZE,default=20"`
}

type StorageConfig struct {
	MinIOEndpoint      string `env:"MINIO_ENDPOINT"`
	MinIOPublicBaseURL string `env:"MINIO_PUBLIC_BASE_URL"`
	MinIOAccessKey     string `env:"MINIO_ACCESS_KEY"`
	MinIOSecretKey     string `env:"MINIO_SECRET_KEY"`
	MinIOBucket        string `env:"MINIO_BUCKET"`
	MinIOUseSSL        bool   `env:"MINIO_USE_SSL"`
}

type CassandraConfig struct {
	Enabled               bool   `env:"CASSANDRA_ENABLED"`
	Hosts                 string `env:"CASSANDRA_HOSTS"`
	Port                  int    `env:"CASSANDRA_PORT,default=9042"`
	Keyspace              string `env:"CASSANDRA_KEYSPACE,default=chat_app"`
	Username              string `env:"CASSANDRA_USERNAME"`
	Password              string `env:"CASSANDRA_PASSWORD"`
	LocalDC               string `env:"CASSANDRA_LOCAL_DC,default=dc1"`
	Consistency           string `env:"CASSANDRA_CONSISTENCY,default=quorum"`
	ReplicationClass      string `env:"CASSANDRA_REPLICATION_CLASS,default=SimpleStrategy"`
	ReplicationFactor     int    `env:"CASSANDRA_REPLICATION_FACTOR,default=1"`
	ConnectTimeoutSeconds int    `env:"CASSANDRA_CONNECT_TIMEOUT_SECONDS,default=10"`
	TimeoutSeconds        int    `env:"CASSANDRA_TIMEOUT_SECONDS,default=10"`
}

type ElasticsearchConfig struct {
	Enabled                  bool   `env:"ELASTICSEARCH_ENABLED"`
	Addresses                string `env:"ELASTICSEARCH_ADDRESSES"`
	Username                 string `env:"ELASTICSEARCH_USERNAME"`
	Password                 string `env:"ELASTICSEARCH_PASSWORD"`
	RoomMessageIndex         string `env:"ELASTICSEARCH_ROOM_MESSAGE_INDEX,default=room_messages_v1"`
	ConnectTimeoutSeconds    int    `env:"ELASTICSEARCH_CONNECT_TIMEOUT_SECONDS,default=10"`
	ResponseHeaderTimeoutSec int    `env:"ELASTICSEARCH_RESPONSE_HEADER_TIMEOUT_SECONDS,default=10"`
}

type SMTPConfig struct {
	Host   string `env:"SMTP_HOST"`
	Port   int    `env:"SMTP_PORT"`
	Secure bool   `env:"SMTP_SECURE"`
	User   string `env:"SMTP_USER"`
	Pass   string `env:"SMTP_PASS"`
	From   string `env:"SMTP_FROM"`
}
