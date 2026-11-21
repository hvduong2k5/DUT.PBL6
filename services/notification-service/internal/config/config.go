package config

import (
	"errors"
	"net"
	"os"
)

type Config struct{ Address, DatabaseURL, LocalToken string }

func Load() (Config, error) {
	c := Config{Address: os.Getenv("NOTIFICATION_ADDRESS"), DatabaseURL: os.Getenv("NOTIFICATION_DATABASE_URL"), LocalToken: os.Getenv("NOTIFICATION_LOCAL_TOKEN")}
	if mode := os.Getenv("APP_MODE"); mode != "" && mode != "local" {
		return c, errors.New("only local mode is available; real identity integration is gated")
	}
	if c.Address == "" {
		c.Address = "127.0.0.1:18017"
	}
	host, _, err := net.SplitHostPort(c.Address)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return c, errors.New("local listener must bind loopback")
	}
	if c.DatabaseURL == "" {
		return c, errors.New("NOTIFICATION_DATABASE_URL required")
	}
	if c.LocalToken != "" && len(c.LocalToken) < 32 {
		return c, errors.New("local token must be at least 32 bytes")
	}
	return c, nil
}
