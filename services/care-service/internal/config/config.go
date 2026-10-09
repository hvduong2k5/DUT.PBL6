package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	HTTPPort    string
	LogLevel    string

	InternalToken string

	DatabaseURL   string
	DBMaxConns    int32
	DBMinConns    int32
	DBMaxConnIdle time.Duration

	KafkaBrokers       []string
	KafkaCareTopic     string
	OutboxPollInterval time.Duration
	OutboxBatchSize    int

	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3Region           string
	S3UseSSL           bool
	S3AutoCreateBucket bool

	MediaUploadURLTTL time.Duration
	MediaAccessURLTTL time.Duration
	MediaOrphanTTL    time.Duration
	ScannerMode       string

	OTLPEndpoint    string
	OTLPInsecure    bool
	OTelServiceName string
}

func Load() (Config, error) {
	cfg := Config{
		Environment: getString("ENVIRONMENT", "development"),
		HTTPPort:    getString("HTTP_PORT", "8006"),
		LogLevel:    getString("LOG_LEVEL", "info"),

		InternalToken: os.Getenv("CARE_INTERNAL_TOKEN"),

		DatabaseURL:   getString("DATABASE_URL", "postgres://care_user:care_password@localhost:5438/care_db?sslmode=disable"),
		DBMaxConns:    int32(getInt("DB_MAX_CONNS", 20)),
		DBMinConns:    int32(getInt("DB_MIN_CONNS", 2)),
		DBMaxConnIdle: getDuration("DB_MAX_CONN_IDLE", 5*time.Minute),

		KafkaBrokers:       splitCSV(getString("KAFKA_BROKERS", "localhost:9092")),
		KafkaCareTopic:     getString("KAFKA_CARE_TOPIC", "care.events.v1"),
		OutboxPollInterval: getDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond),
		OutboxBatchSize:    getInt("OUTBOX_BATCH_SIZE", 50),

		S3Endpoint:         getString("S3_ENDPOINT", "localhost:9010"),
		S3AccessKey:        getString("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getString("S3_SECRET_KEY", "minioadminpassword"),
		S3Bucket:           getString("S3_BUCKET", "care-private"),
		S3Region:           getString("S3_REGION", "us-east-1"),
		S3UseSSL:           getBool("S3_USE_SSL", false),
		S3AutoCreateBucket: getBool("S3_AUTO_CREATE_BUCKET", true),

		MediaUploadURLTTL: getDuration("MEDIA_UPLOAD_URL_TTL", 15*time.Minute),
		MediaAccessURLTTL: getDuration("MEDIA_ACCESS_URL_TTL", 10*time.Minute),
		MediaOrphanTTL:    getDuration("MEDIA_ORPHAN_TTL", 24*time.Hour),
		ScannerMode:       strings.ToLower(getString("SCANNER_MODE", "noop")),

		OTLPEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTLPInsecure:    getBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		OTelServiceName: getString("OTEL_SERVICE_NAME", "care-service"),
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	var validationErrors []error
	if strings.TrimSpace(c.InternalToken) == "" {
		validationErrors = append(validationErrors, errors.New("CARE_INTERNAL_TOKEN is required"))
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		validationErrors = append(validationErrors, errors.New("DATABASE_URL is required"))
	}
	if c.DBMaxConns < 1 || c.DBMinConns < 0 || c.DBMinConns > c.DBMaxConns {
		validationErrors = append(validationErrors, errors.New("invalid PostgreSQL pool size"))
	}
	if len(c.KafkaBrokers) == 0 || c.KafkaCareTopic == "" {
		validationErrors = append(validationErrors, errors.New("Kafka broker and care topic are required"))
	}
	if c.S3Endpoint == "" || c.S3AccessKey == "" || c.S3SecretKey == "" || c.S3Bucket == "" {
		validationErrors = append(validationErrors, errors.New("S3 endpoint, credentials, and bucket are required"))
	}
	if c.MediaUploadURLTTL <= 0 || c.MediaAccessURLTTL <= 0 || c.MediaOrphanTTL <= 0 {
		validationErrors = append(validationErrors, errors.New("media TTL values must be positive"))
	}
	if c.ScannerMode != "noop" {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported SCANNER_MODE %q", c.ScannerMode))
	}
	if strings.EqualFold(c.Environment, "production") && c.ScannerMode == "noop" {
		validationErrors = append(validationErrors, errors.New("SCANNER_MODE=noop is forbidden in production"))
	}
	return errors.Join(validationErrors...)
}

func getString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
