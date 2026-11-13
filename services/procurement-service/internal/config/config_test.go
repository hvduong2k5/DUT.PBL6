package config_test

import (
	"testing"

	"github.com/omamx/procurement-service/internal/config"
)

func TestLoadRequiresInternalToken(t *testing.T) {
	t.Setenv("PROCUREMENT_INTERNAL_TOKEN", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected missing internal token to fail")
	}
}

func TestLoadUsesDefaultsAndConfiguredToken(t *testing.T) {
	t.Setenv("PROCUREMENT_INTERNAL_TOKEN", "test-secret")
	t.Setenv("PROCUREMENT_HTTP_ADDRESS", "")
	t.Setenv("PROCUREMENT_DATABASE_URL", "")
	t.Setenv("KAFKA_BROKERS", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":8008" || cfg.DatabaseURL != "postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable" || cfg.InternalToken != "test-secret" || cfg.KafkaBrokers != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadReadsKafkaBrokers(t *testing.T) {
	t.Setenv("PROCUREMENT_INTERNAL_TOKEN", "test-secret")
	t.Setenv("KAFKA_BROKERS", " kafka:9092, kafka-backup:9092 ")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KafkaBrokers != "kafka:9092, kafka-backup:9092" {
		t.Fatalf("KafkaBrokers = %q", cfg.KafkaBrokers)
	}
}
