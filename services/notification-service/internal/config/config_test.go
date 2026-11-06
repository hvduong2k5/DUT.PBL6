package config

import "testing"

func TestLocalBoundary(t *testing.T) {
	t.Setenv("APP_MODE", "local")
	t.Setenv("NOTIFICATION_DATABASE_URL", "postgres://example/notification")
	t.Setenv("NOTIFICATION_LOCAL_TOKEN", "")
	t.Setenv("NOTIFICATION_ADDRESS", "")
	if _, e := Load(); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ key, value string }{{"APP_MODE", "production"}, {"NOTIFICATION_ADDRESS", "0.0.0.0:8017"}, {"NOTIFICATION_LOCAL_TOKEN", "short"}, {"NOTIFICATION_DATABASE_URL", ""}} {
		t.Run(tc.key, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, e := Load(); e == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
}
