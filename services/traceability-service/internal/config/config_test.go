package config

import (
	"strings"
	"testing"
)

func TestLoadBoundaries(t *testing.T) {
	tests := []struct {
		name string
		set  map[string]string
		want string
	}{
		{"sandbox", nil, ""},
		{"explicit mode", map[string]string{"APP_MODE": ""}, "APP_MODE is required"},
		{"production closed", map[string]string{"APP_MODE": "production"}, "production is not enabled"},
		{"external bind closed", map[string]string{"HTTP_ADDR": "0.0.0.0:8003"}, "loopback"},
		{"container opt in", map[string]string{"HTTP_ADDR": "0.0.0.0:8003", "SANDBOX_ISOLATED_CONTAINER": "true"}, ""},
		{"DNS bind refused", map[string]string{"HTTP_ADDR": "localhost:8003"}, "numeric IP"},
		{"port range", map[string]string{"HTTP_ADDR": "127.0.0.1:0"}, "65535"},
		{"invalid container flag", map[string]string{"SANDBOX_ISOLATED_CONTAINER": "maybe"}, "boolean"},
		{"integration sandbox flag", map[string]string{"APP_MODE": "integration", "SANDBOX_ISOLATED_CONTAINER": "true"}, "only allowed"},
		{"no DB", map[string]string{"DATABASE_URL": ""}, "DATABASE_URL"},
		{"no DB name", map[string]string{"DATABASE_URL": "postgres://localhost/"}, "DATABASE_URL"},
		{"unsafe public HTTP", map[string]string{"PUBLIC_WEB_BASE_URL": "http://example.org"}, "HTTPS"},
		{"public URL credentials", map[string]string{"PUBLIC_WEB_BASE_URL": "https://user:secret@example.org"}, "credentials"},
		{"public URL query", map[string]string{"PUBLIC_WEB_BASE_URL": "https://example.org?token=x"}, "query"},
		{"integration HTTPS", map[string]string{"APP_MODE": "integration", "PUBLIC_WEB_BASE_URL": "http://localhost:3000"}, "HTTPS"},
		{"integration configured", map[string]string{"APP_MODE": "integration", "PUBLIC_WEB_BASE_URL": "https://example.org"}, ""},
		{"bounded shutdown", map[string]string{"SHUTDOWN_TIMEOUT": "2h"}, "SHUTDOWN_TIMEOUT"},
		{"video hosts", map[string]string{"VIDEO_ALLOWED_HOSTS": "cdn.example.org,video.example.org"}, ""},
		{"video URL rejected", map[string]string{"VIDEO_ALLOWED_HOSTS": "https://cdn.example.org/path"}, "VIDEO_ALLOWED_HOSTS"},
		{"video credentials rejected", map[string]string{"VIDEO_ALLOWED_HOSTS": "user:secret@cdn.example.org"}, "VIDEO_ALLOWED_HOSTS"},
		{"duplicate host rejected", map[string]string{"VIDEO_ALLOWED_HOSTS": "cdn.example.org,CDN.EXAMPLE.ORG"}, "VIDEO_ALLOWED_HOSTS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{"APP_MODE": "sandbox", "DATABASE_URL": "postgres://test:secret@localhost:5432/trace_test", "PUBLIC_WEB_BASE_URL": "http://localhost:3000"}
			for k, v := range tt.set {
				env[k] = v
			}
			cfg, err := Load(func(k string) string { return env[k] })
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				if cfg.ListenAddress == "" {
					t.Fatal("missing listener default")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want %q, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error leaked credential")
			}
		})
	}
}
