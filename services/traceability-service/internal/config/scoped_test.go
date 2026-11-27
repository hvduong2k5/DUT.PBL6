package config

import (
	"strings"
	"testing"
)

func TestScopedCapabilitiesRequireExplicitSecrets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		valid  bool
	}{
		{"off by default", map[string]string{}, true},
		{"scope without key", map[string]string{"TRACEABILITY_SCOPED_API_ENABLED": "true"}, false},
		{"scope with key", map[string]string{"TRACEABILITY_SCOPED_API_ENABLED": "true", "TRACEABILITY_CURSOR_HMAC_KEY": strings.Repeat("k", 32)}, true},
		{"resolver without credential", map[string]string{"TRACEABILITY_ORDER_RESOLVER_ENABLED": "true"}, false},
		{"resolver independent", map[string]string{"TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32)}, true},
		{"resolver reused credential", map[string]string{"TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32), "TRACEABILITY_CATALOG_TOKEN": strings.Repeat("t", 32)}, false},
		{"invalid flag", map[string]string{"TRACEABILITY_SCOPED_API_ENABLED": "maybe"}, false},
		{"integration resolver requires TLS", map[string]string{"APP_MODE": "integration", "PUBLIC_WEB_BASE_URL": "https://shop.example.test", "TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32)}, false},
		{"integration resolver TLS configured", map[string]string{"APP_MODE": "integration", "PUBLIC_WEB_BASE_URL": "https://shop.example.test", "TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32), "GRPC_TLS_CERT_FILE": "server.pem", "GRPC_TLS_KEY_FILE": "server-key.pem"}, true},
		{"nonloopback sandbox RPC", map[string]string{"TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32), "GRPC_ADDR": "0.0.0.0:9003"}, false},
		{"reuse IAM secret", map[string]string{"TRACEABILITY_ORDER_RESOLVER_ENABLED": "true", "TRACEABILITY_ORDER_RESOLVER_TOKEN": strings.Repeat("t", 32), "TRACEABILITY_IDENTITY_TOKEN": strings.Repeat("t", 32)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"APP_MODE": "sandbox", "DATABASE_URL": "postgres://localhost/test", "PUBLIC_WEB_BASE_URL": "http://localhost:3000"}
			for k, v := range tc.values {
				env[k] = v
			}
			_, err := Load(func(k string) string { return env[k] })
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
