package config

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"strconv"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	DBMaxConns  int32
	LogLevel    zerolog.Level
}

func LoadConfig() (Config, error) {
	c := Config{HTTPPort: env("HTTP_PORT", "8010"), DatabaseURL: os.Getenv("DATABASE_URL"), DBMaxConns: 5}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	p, err := strconv.Atoi(c.HTTPPort)
	if err != nil || p < 1 || p > 65535 {
		return c, fmt.Errorf("HTTP_PORT must be 1..65535")
	}
	n, err := strconv.Atoi(env("DB_MAX_CONNS", "5"))
	if err != nil || n < 1 || n > 50 {
		return c, fmt.Errorf("DB_MAX_CONNS must be 1..50")
	}
	c.DBMaxConns = int32(n)
	c.LogLevel, err = zerolog.ParseLevel(env("LOG_LEVEL", "info"))
	if err != nil {
		return c, fmt.Errorf("invalid LOG_LEVEL")
	}
	return c, nil
}
func env(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
