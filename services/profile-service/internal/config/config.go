package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	IdentityToken      string
	AuthMode           string
	OIDCIssuer         string
	OIDCAudience       string
	OIDCJWKSURL        string
	OIDCAllowLocalHTTP bool
	HTTPPort           string
	GRPCPort           string
	LogLevel           string

	// Database
	DatabaseURL  string
	DBMaxConns   int
	DBMinConns   int
	DBMaxIdleSec int

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	L1CacheCap    int

	// Vault
	VaultAddr       string
	VaultToken      string
	VaultKeyName    string
	DEKCacheTTL     time.Duration
	UseVaultMockDev bool

	// Kafka
	KafkaBrokers       string
	TopicProfileEvents string

	// Identity event consumption (customer provisioning sync)
	IdentityEventsTopic   string
	IdentityConsumerGroup string

	// Workers
	OutboxPollInterval  time.Duration
	OutboxBatchSize     int
	ComplianceCheckCron string
}

func LoadConfig() *Config {
	return &Config{
		IdentityToken:      getEnv("PROFILE_INTERNAL_TOKEN", ""),
		AuthMode:           getEnv("PROFILE_AUTH_MODE", "internal"),
		OIDCIssuer:         getEnv("PROFILE_OIDC_ISSUER", ""),
		OIDCAudience:       getEnv("PROFILE_OIDC_AUDIENCE", ""),
		OIDCJWKSURL:        getEnv("PROFILE_OIDC_JWKS_URL", ""),
		OIDCAllowLocalHTTP: getEnvAsBool("PROFILE_OIDC_ALLOW_LOCAL_HTTP", false),
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		GRPCPort:           getEnv("GRPC_PORT", "50051"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),

		DatabaseURL:  getEnv("DATABASE_URL", "postgres://omamx_user:omamx_password@localhost:5432/profile_db?sslmode=disable"),
		DBMaxConns:   getEnvAsInt("DB_MAX_CONNS", 50),
		DBMinConns:   getEnvAsInt("DB_MIN_CONNS", 5),
		DBMaxIdleSec: getEnvAsInt("DB_MAX_IDLE_SEC", 300),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		L1CacheCap:    getEnvAsInt("L1_CACHE_CAP", 5000),

		VaultAddr:       getEnv("VAULT_ADDR", "http://localhost:8200"),
		VaultToken:      getEnv("VAULT_TOKEN", "root"),
		VaultKeyName:    getEnv("VAULT_KEY_NAME", "profile-pii-kek"),
		DEKCacheTTL:     getEnvAsDuration("DEK_CACHE_TTL_MIN", 5) * time.Minute,
		UseVaultMockDev: getEnvAsBool("USE_VAULT_MOCK_DEV", false),

		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		TopicProfileEvents: getEnv("TOPIC_PROFILE_EVENTS", "profile.events.v1"),

		IdentityEventsTopic:   getEnv("IDENTITY_EVENTS_TOPIC", "identity.events.v1"),
		IdentityConsumerGroup: getEnv("IDENTITY_CONSUMER_GROUP", "profile-identity-sync"),

		OutboxPollInterval:  getEnvAsDuration("OUTBOX_POLL_INTERVAL_MS", 500) * time.Millisecond,
		OutboxBatchSize:     getEnvAsInt("OUTBOX_BATCH_SIZE", 50),
		ComplianceCheckCron: getEnv("COMPLIANCE_CHECK_CRON", "0 1 * * *"), // 1 AM daily
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsDuration(key string, defaultVal int) time.Duration {
	return time.Duration(getEnvAsInt(key, defaultVal))
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
