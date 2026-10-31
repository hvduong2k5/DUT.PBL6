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
	FulfillmentV2Topic string
	FulfillmentV2Group string
	OrderGRPCEndpoint  string
	OrderInternalToken string
	DefaultCarrier     string
	DefaultService     string
	OutboxPollInterval time.Duration
	ShutdownTimeout    time.Duration
	DatabaseTimeout    time.Duration
	V2Enabled          bool
	PackingV2Enabled   bool
	GHNEnabled         bool
	GHNBaseURL         string
	GHNToken           string
	GHNShopID          int64
	GHNFromDistrictID  int
	GHNFromWardCode    string
	GHNWebhookSecret   string
	GHNRequestTimeout  time.Duration
	GHNRecoveryTimeout time.Duration
	GHNQuoteTTL        time.Duration
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
		FulfillmentV2Topic: env("SHIPPING_FULFILLMENT_V2_TOPIC", "fulfillment.events.v2"),
		FulfillmentV2Group: env("SHIPPING_FULFILLMENT_V2_GROUP", "shipping-service-v2"),
		OrderGRPCEndpoint:  strings.TrimSpace(os.Getenv("SHIPPING_ORDER_GRPC_ENDPOINT")),
		OrderInternalToken: strings.TrimSpace(os.Getenv("SHIPPING_ORDER_INTERNAL_TOKEN")),
		DefaultCarrier:     env("SHIPPING_DEFAULT_CARRIER", "FAKE"),
		DefaultService:     env("SHIPPING_DEFAULT_SERVICE", "STANDARD"),
		OutboxPollInterval: durationEnv("SHIPPING_OUTBOX_POLL_INTERVAL", time.Second),
		ShutdownTimeout:    durationEnv("SHIPPING_SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseTimeout:    durationEnv("SHIPPING_DATABASE_TIMEOUT", 5*time.Second),
		V2Enabled:          boolEnv("SHIPPING_V2_ENABLED", false),
		PackingV2Enabled:   boolEnv("SHIPPING_PACKING_V2_ENABLED", false),
		GHNEnabled:         boolEnv("SHIPPING_GHN_ENABLED", false),
		GHNBaseURL:         env("SHIPPING_GHN_BASE_URL", "https://dev-online-gateway.ghn.vn"),
		GHNToken:           strings.TrimSpace(os.Getenv("SHIPPING_GHN_TOKEN")),
		GHNShopID:          int64Env("SHIPPING_GHN_SHOP_ID", 0),
		GHNFromDistrictID:  intEnv("SHIPPING_GHN_FROM_DISTRICT_ID", 0),
		GHNFromWardCode:    strings.TrimSpace(os.Getenv("SHIPPING_GHN_FROM_WARD_CODE")),
		GHNWebhookSecret:   strings.TrimSpace(os.Getenv("SHIPPING_GHN_WEBHOOK_SECRET")),
		GHNRequestTimeout:  durationEnv("SHIPPING_GHN_REQUEST_TIMEOUT", 10*time.Second),
		GHNRecoveryTimeout: durationEnv("SHIPPING_GHN_RECONCILE_TIMEOUT", 3*time.Second),
		GHNQuoteTTL:        durationEnv("SHIPPING_GHN_QUOTE_TTL", 15*time.Minute),
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
	if cfg.GHNEnabled {
		if cfg.GHNToken == "" || cfg.GHNShopID <= 0 || cfg.GHNFromDistrictID <= 0 || cfg.GHNFromWardCode == "" {
			return Config{}, fmt.Errorf("SHIPPING_GHN_TOKEN, SHIPPING_GHN_SHOP_ID, SHIPPING_GHN_FROM_DISTRICT_ID and SHIPPING_GHN_FROM_WARD_CODE are required when GHN is enabled")
		}
		if cfg.GHNWebhookSecret == "" {
			return Config{}, fmt.Errorf("SHIPPING_GHN_WEBHOOK_SECRET is required when GHN is enabled")
		}
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

func intEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func int64Env(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}
