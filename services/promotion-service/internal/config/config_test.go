package config

import "testing"

func TestLoadRequiresInternalToken(t *testing.T) {
	t.Setenv("PROMOTION_INTERNAL_TOKEN", " ")
	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without PROMOTION_INTERNAL_TOKEN")
	}
}

func TestLoadDefaultsAndTrimsValues(t *testing.T) {
	t.Setenv("PROMOTION_INTERNAL_TOKEN", " local-token ")
	t.Setenv("PROMOTION_HTTP_ADDRESS", " ")
	t.Setenv("PROMOTION_ORDER_EVENTS_ENABLED", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InternalToken != "local-token" {
		t.Fatalf("InternalToken = %q", cfg.InternalToken)
	}
	if cfg.HTTPAddress != ":8017" || cfg.GRPCAddress != ":9017" {
		t.Fatalf("unexpected default addresses: %q, %q", cfg.HTTPAddress, cfg.GRPCAddress)
	}
	if cfg.OrderEventsEnabled {
		t.Fatal("OrderEventsEnabled should default to false")
	}
}

func TestLoadRequiresKafkaWhenOrderEventsEnabled(t *testing.T) {
	t.Setenv("PROMOTION_INTERNAL_TOKEN", "token")
	t.Setenv("PROMOTION_ORDER_EVENTS_ENABLED", "true")
	t.Setenv("PROMOTION_KAFKA_BROKERS", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without Kafka brokers")
	}
}

func TestLoadAcceptsKafkaConfiguration(t *testing.T) {
	t.Setenv("PROMOTION_INTERNAL_TOKEN", "token")
	t.Setenv("PROMOTION_ORDER_EVENTS_ENABLED", "true")
	t.Setenv("PROMOTION_KAFKA_BROKERS", "localhost:9092")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.OrderEventsEnabled || cfg.OrderEventsTopic != "order.events.v3" || cfg.OrderConsumerGroup != "promotion-order-v3" {
		t.Fatalf("unexpected Order event defaults: %+v", cfg)
	}
}

func TestLoadPromotionStaffJWTConfiguration(t *testing.T) {
	t.Setenv("PROMOTION_INTERNAL_TOKEN", "token")
	t.Setenv("PROMOTION_STAFF_JWT_ISSUER", " https://keycloak.example/realms/omama ")
	t.Setenv("PROMOTION_STAFF_JWT_AUDIENCE", " omama-api ")
	t.Setenv("PROMOTION_STAFF_JWT_PUBLIC_KEY_PATH", " /run/secrets/identity-public.pem ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.StaffJWTIssuer != "https://keycloak.example/realms/omama" || cfg.StaffJWTAudience != "omama-api" || cfg.StaffJWTPublicKeyPath != "/run/secrets/identity-public.pem" {
		t.Fatalf("unexpected Promotion JWT config: %+v", cfg)
	}
}
