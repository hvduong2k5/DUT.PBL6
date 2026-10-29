package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress        string
	GRPCAddress        string
	DatabaseURL        string
	InternalToken      string
	FakeWebhookSecret  string
	KafkaBrokers       []string
	FulfillmentTopic   string
	FulfillmentGroup   string
	OrderGRPCEndpoint  string
	OrderInternalToken string
	DefaultCarrier     string
	DefaultService     string
	OutboxPollInterval time.Duration
	ShutdownTimeout    time.Duration
	DatabaseTimeout    time.Duration
	V2Enabled          bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:        env("SHIPPING_HTTP_ADDRESS", ":8012"),
		GRPCAddress:        env("SHIPPING_GRPC_ADDRESS", ":9012"),
		DatabaseURL:        env("SHIPPING_DATABASE_URL", "postgres://shipping:shipping@localhost:5432/shipping_db?sslmode=disable"),
		InternalToken:      strings.TrimSpace(os.Getenv("SHIPPING_INTERNAL_TOKEN")),
		FakeWebhookSecret:  strings.TrimSpace(os.Getenv("SHIPPING_FAKE_WEBHOOK_SECRET")),
		KafkaBrokers:       csvEnv("SHIPPING_KAFKA_BROKERS", "localhost:9092"),
		FulfillmentTopic:   env("SHIPPING_FULFILLMENT_TOPIC", "fulfillment.events.v1"),
		FulfillmentGroup:   env("SHIPPING_FULFILLMENT_GROUP", "shipping-service-v1"),
		OrderGRPCEndpoint:  strings.TrimSpace(os.Getenv("SHIPPING_ORDER_GRPC_ENDPOINT")),
		OrderInternalToken: strings.TrimSpace(os.Getenv("SHIPPING_ORDER_INTERNAL_TOKEN")),
		DefaultCarrier:     env("SHIPPING_DEFAULT_CARRIER", "FAKE"),
		DefaultService:     env("SHIPPING_DEFAULT_SERVICE", "STANDARD"),
		OutboxPollInterval: durationEnv("SHIPPING_OUTBOX_POLL_INTERVAL", time.Second),
		ShutdownTimeout:    durationEnv("SHIPPING_SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseTimeout:    durationEnv("SHIPPING_DATABASE_TIMEOUT", 5*time.Second),
		V2Enabled:          boolEnv("SHIPPING_V2_ENABLED", false),
	}
	if cfg.InternalToken == "" {
		return Config{}, fmt.Errorf("SHIPPING_INTERNAL_TOKEN is required")
	}
	if cfg.FakeWebhookSecret == "" {
		return Config{}, fmt.Errorf("SHIPPING_FAKE_WEBHOOK_SECRET is required")
	}
	if cfg.OrderGRPCEndpoint != "" && cfg.OrderInternalToken == "" {
		return Config{}, fmt.Errorf("SHIPPING_ORDER_INTERNAL_TOKEN is required when SHIPPING_ORDER_GRPC_ENDPOINT is set")
	}
	return cfg, nil
}

func boolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func csvEnv(name, fallback string) []string {
	raw := env(name, fallback)
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if value, err := time.ParseDuration(raw); err == nil {
		return value
	}
	return fallback
}
