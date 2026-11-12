package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress           string
	GRPCAddress           string
	DatabaseURL           string
	InternalToken         string
	StaffJWTIssuer        string
	StaffJWTAudience      string
	StaffJWTPublicKeyPath string
	DBTimeout             time.Duration
	ShutdownWait          time.Duration
	OrderEventsEnabled    bool
	KafkaBrokers          string
	OrderEventsTopic      string
	OrderEventsDLQTopic   string
	OrderConsumerGroup    string
	PromotionEventsTopic  string
	LoyaltyOrderToken     string
	LoyaltyRefundToken    string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:           env("PROMOTION_HTTP_ADDRESS", ":8017"),
		GRPCAddress:           env("PROMOTION_GRPC_ADDRESS", ":9017"),
		DatabaseURL:           env("PROMOTION_DATABASE_URL", "postgres://promotion:promotion@localhost:5447/promotion_db?sslmode=disable"),
		InternalToken:         strings.TrimSpace(os.Getenv("PROMOTION_INTERNAL_TOKEN")),
		StaffJWTIssuer:        strings.TrimSpace(os.Getenv("PROMOTION_STAFF_JWT_ISSUER")),
		StaffJWTAudience:      env("PROMOTION_STAFF_JWT_AUDIENCE", "omama-api"),
		StaffJWTPublicKeyPath: strings.TrimSpace(os.Getenv("PROMOTION_STAFF_JWT_PUBLIC_KEY_PATH")),
		DBTimeout:             5 * time.Second,
		ShutdownWait:          10 * time.Second,
		OrderEventsEnabled:    strings.EqualFold(env("PROMOTION_ORDER_EVENTS_ENABLED", "false"), "true"),
		KafkaBrokers:          strings.TrimSpace(os.Getenv("PROMOTION_KAFKA_BROKERS")),
		OrderEventsTopic:      env("PROMOTION_ORDER_EVENTS_TOPIC", "order.events.v3"),
		OrderEventsDLQTopic:   env("PROMOTION_ORDER_EVENTS_DLQ_TOPIC", "promotion.order.v3.dlq"),
		OrderConsumerGroup:    env("PROMOTION_ORDER_CONSUMER_GROUP", "promotion-order-v3"),
		PromotionEventsTopic:  env("PROMOTION_EVENTS_TOPIC", "promotion.events.v1"),
		LoyaltyOrderToken:     strings.TrimSpace(os.Getenv("PROMOTION_LOYALTY_ORDER_TOKEN")),
		LoyaltyRefundToken:    strings.TrimSpace(os.Getenv("PROMOTION_LOYALTY_REFUND_TOKEN")),
	}
	if cfg.InternalToken == "" {
		return Config{}, fmt.Errorf("PROMOTION_INTERNAL_TOKEN is required")
	}
	if cfg.OrderEventsEnabled && cfg.KafkaBrokers == "" {
		return Config{}, fmt.Errorf("PROMOTION_KAFKA_BROKERS is required when order event consumption is enabled")
	}
	if cfg.OrderEventsEnabled && (cfg.OrderEventsTopic == "" || cfg.OrderConsumerGroup == "" || cfg.OrderEventsDLQTopic == "" || cfg.PromotionEventsTopic == "") {
		return Config{}, fmt.Errorf("Kafka topics and consumer group are required when order event consumption is enabled")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
