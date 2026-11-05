package config

import (
	"github.com/stretchr/testify/require"
	"testing"
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
	for _, tc := range []struct{ key, value string }{{"HTTP_PORT", "bad"}, {"HTTP_PORT", "0"}, {"DB_MAX_CONNS", "0"}, {"DB_MAX_CONNS", "1000"}, {"LOG_LEVEL", "bad"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) { t.Setenv(tc.key, tc.value); _, err := LoadConfig(); require.Error(t, err) })
	}
}
