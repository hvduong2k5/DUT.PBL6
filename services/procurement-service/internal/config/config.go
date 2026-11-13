package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress   string
	DatabaseURL   string
	InternalToken string
	KafkaBrokers  string
	DBTimeout     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:   env("PROCUREMENT_HTTP_ADDRESS", ":8008"),
		DatabaseURL:   env("PROCUREMENT_DATABASE_URL", "postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable"),
		InternalToken: strings.TrimSpace(os.Getenv("PROCUREMENT_INTERNAL_TOKEN")),
		KafkaBrokers:  strings.TrimSpace(os.Getenv("KAFKA_BROKERS")),
		DBTimeout:     5 * time.Second,
	}
	if cfg.InternalToken == "" {
		return Config{}, fmt.Errorf("PROCUREMENT_INTERNAL_TOKEN is required")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
