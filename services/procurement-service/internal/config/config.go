package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress        string
	DatabaseURL        string
	OIDCIssuer         string
	OIDCAudience       string
	OIDCJWKSURL        string
	OIDCAllowLocalHTTP bool
	KafkaBrokers       string
	DBTimeout          time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:        env("PROCUREMENT_HTTP_ADDRESS", ":8008"),
		DatabaseURL:        env("PROCUREMENT_DATABASE_URL", "postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable"),
		OIDCIssuer:         strings.TrimRight(strings.TrimSpace(os.Getenv("PROCUREMENT_OIDC_ISSUER")), "/"),
		OIDCAudience:       env("PROCUREMENT_OIDC_AUDIENCE", "omama-api"),
		OIDCJWKSURL:        strings.TrimSpace(os.Getenv("PROCUREMENT_OIDC_JWKS_URL")),
		OIDCAllowLocalHTTP: strings.EqualFold(strings.TrimSpace(os.Getenv("PROCUREMENT_OIDC_ALLOW_LOCAL_HTTP")), "true"),
		KafkaBrokers:       strings.TrimSpace(os.Getenv("KAFKA_BROKERS")),
		DBTimeout:          5 * time.Second,
	}
	if cfg.OIDCIssuer == "" {
		return Config{}, fmt.Errorf("PROCUREMENT_OIDC_ISSUER is required")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
