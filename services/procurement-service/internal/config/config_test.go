package config_test

import (
	"testing"

	"github.com/omamx/procurement-service/internal/config"
)

func TestLoadRequiresIdentityJWTConfiguration(t *testing.T) {
	t.Setenv("PROCUREMENT_OIDC_ISSUER", "")
	t.Setenv("PROCUREMENT_OIDC_AUDIENCE", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected missing OIDC configuration to fail")
	}
}

func TestLoadUsesDefaultsAndIdentityConfiguration(t *testing.T) {
	t.Setenv("PROCUREMENT_OIDC_ISSUER", "https://keycloak.example/realms/omama/")
	t.Setenv("PROCUREMENT_OIDC_AUDIENCE", "omama-api")
	t.Setenv("PROCUREMENT_HTTP_ADDRESS", "")
	t.Setenv("PROCUREMENT_DATABASE_URL", "")
	t.Setenv("KAFKA_BROKERS", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":8008" || cfg.DatabaseURL != "postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable" || cfg.OIDCIssuer != "https://keycloak.example/realms/omama" || cfg.OIDCAudience != "omama-api" || cfg.KafkaBrokers != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadReadsKafkaBrokers(t *testing.T) {
	t.Setenv("PROCUREMENT_OIDC_ISSUER", "https://keycloak.example/realms/omama")
	t.Setenv("PROCUREMENT_OIDC_AUDIENCE", "omama-api")
	t.Setenv("KAFKA_BROKERS", " kafka:9092, kafka-backup:9092 ")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KafkaBrokers != "kafka:9092, kafka-backup:9092" {
		t.Fatalf("KafkaBrokers = %q", cfg.KafkaBrokers)
	}
}
