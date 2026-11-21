package config

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	LogLevel    zerolog.Level

	DBMaxConns       int32
	DBMinConns       int32
	DBConnLifetime   time.Duration
	DBConnectTimeout time.Duration

	RequestTimeout   time.Duration
	ReadinessTimeout time.Duration
	ShutdownTimeout  time.Duration
	MaxBodyBytes     int64

	// Staff token verification. Without a public key and issuer the admin API
	// stays closed (401) and the service logs why.
	JWTIssuer        string
	JWTPublicKeyFile string
	StaffAudience    string
}

func LoadConfig() (Config, error) {
	c := Config{
		HTTPPort:         env("HTTP_PORT", "8010"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTIssuer:        os.Getenv("CONTENT_JWT_ISSUER"),
		JWTPublicKeyFile: os.Getenv("CONTENT_JWT_PUBLIC_KEY"),
		StaffAudience:    env("CONTENT_STAFF_AUDIENCE", "omama-api"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	p, err := strconv.Atoi(c.HTTPPort)
	if err != nil || p < 1 || p > 65535 {
		return c, fmt.Errorf("HTTP_PORT must be 1..65535")
	}
	max, err := intEnv("DB_MAX_CONNS", 5, 1, 50)
	if err != nil {
		return c, err
	}
	min, err := intEnv("DB_MIN_CONNS", 0, 0, 50)
	if err != nil {
		return c, err
	}
	if min > max {
		return c, fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS")
	}
	c.DBMaxConns, c.DBMinConns = int32(max), int32(min)
	if c.DBConnLifetime, err = durationEnv("DB_CONN_LIFETIME", 30*time.Minute, time.Minute, 24*time.Hour); err != nil {
		return c, err
	}
	if c.DBConnectTimeout, err = durationEnv("DB_CONNECT_TIMEOUT", 3*time.Second, time.Second, time.Minute); err != nil {
		return c, err
	}
	if c.RequestTimeout, err = durationEnv("HTTP_REQUEST_TIMEOUT", 10*time.Second, time.Second, time.Minute); err != nil {
		return c, err
	}
	if c.ReadinessTimeout, err = durationEnv("READINESS_TIMEOUT", time.Second, 100*time.Millisecond, 30*time.Second); err != nil {
		return c, err
	}
	if c.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second, time.Second, 5*time.Minute); err != nil {
		return c, err
	}
	body, err := intEnv("MAX_BODY_BYTES", 1<<20, 1<<10, 16<<20)
	if err != nil {
		return c, err
	}
	c.MaxBodyBytes = int64(body)
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
func intEnv(key string, fallback, min, max int) (int, error) {
	n, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s must be %d..%d", key, min, max)
	}
	return n, nil
}
func durationEnv(key string, fallback, min, max time.Duration) (time.Duration, error) {
	d, err := time.ParseDuration(env(key, fallback.String()))
	if err != nil || d < min || d > max {
		return 0, fmt.Errorf("%s must be a duration between %s and %s", key, min, max)
	}
	return d, nil
}
