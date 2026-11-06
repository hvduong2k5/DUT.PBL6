package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress   string
	GRPCAddress   string
	DatabaseURL   string
	InternalToken string
	DBTimeout     time.Duration
	ShutdownWait  time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:   env("PROMOTION_HTTP_ADDRESS", ":8017"),
		GRPCAddress:   env("PROMOTION_GRPC_ADDRESS", ":9017"),
		DatabaseURL:   env("PROMOTION_DATABASE_URL", "postgres://promotion:promotion@localhost:5447/promotion_db?sslmode=disable"),
		InternalToken: strings.TrimSpace(os.Getenv("PROMOTION_INTERNAL_TOKEN")),
		DBTimeout:     5 * time.Second,
		ShutdownWait:  10 * time.Second,
	}
	if cfg.InternalToken == "" {
		return Config{}, fmt.Errorf("PROMOTION_INTERNAL_TOKEN is required")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
