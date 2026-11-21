package config

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := LoadConfig()
	require.Error(t, err)
	t.Setenv("DATABASE_URL", "postgres://localhost/content")
	t.Setenv("HTTP_PORT", "8010")
	t.Setenv("DB_MAX_CONNS", "5")
	t.Setenv("LOG_LEVEL", "info")
	c, err := LoadConfig()
	require.NoError(t, err)
	require.EqualValues(t, 5, c.DBMaxConns)
	require.Equal(t, 10*time.Second, c.RequestTimeout)
	require.Equal(t, time.Second, c.ReadinessTimeout)
	require.Equal(t, 15*time.Second, c.ShutdownTimeout)
	require.EqualValues(t, 1<<20, c.MaxBodyBytes)
	require.Equal(t, "omama-api", c.StaffAudience)
	for _, tc := range []struct{ key, value string }{
		{"HTTP_PORT", "bad"}, {"HTTP_PORT", "0"}, {"DB_MAX_CONNS", "0"}, {"DB_MAX_CONNS", "1000"}, {"LOG_LEVEL", "bad"},
		{"DB_MIN_CONNS", "6"}, {"DB_MIN_CONNS", "-1"}, {"HTTP_REQUEST_TIMEOUT", "soon"}, {"HTTP_REQUEST_TIMEOUT", "0s"},
		{"READINESS_TIMEOUT", "10m"}, {"SHUTDOWN_TIMEOUT", "0"}, {"DB_CONN_LIFETIME", "1s"}, {"DB_CONNECT_TIMEOUT", "1h"}, {"MAX_BODY_BYTES", "10"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) { t.Setenv(tc.key, tc.value); _, err := LoadConfig(); require.Error(t, err) })
	}
	t.Setenv("HTTP_REQUEST_TIMEOUT", "3s")
	t.Setenv("MAX_BODY_BYTES", "2048")
	c, err = LoadConfig()
	require.NoError(t, err)
	require.Equal(t, 3*time.Second, c.RequestTimeout)
	require.EqualValues(t, 2048, c.MaxBodyBytes)
}
