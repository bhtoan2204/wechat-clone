package config

type Config struct {
	ConsulConfig ConsulConfig
	HTTP         HTTPConfig
	AuthConfig   AuthConfig
	RedisConfig  RedisConfig
}

type ConsulConfig struct {
	Address    string `env:"CONSUL_ADDRESS"`
	Scheme     string `env:"CONSUL_SCHEME"`
	DataCenter string `env:"CONSUL_DATA_CENTER"`
	Token      string `env:"CONSUL_TOKEN"`
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT"`
}

type AuthConfig struct {
	AccessPublicKey string `env:"AUTH_ACCESS_PUBLIC_KEY"`
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
