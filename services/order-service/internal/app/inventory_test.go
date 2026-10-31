package app

import (
	"context"
	"testing"
)

func TestInventoryProfileCannotDowngrade(t *testing.T) {
	keys := []string{"SEPAY_INVENTORY_GRPC", "SEPAY_INVENTORY_PROFILE", "SEPAY_INVENTORY_TOKEN_FILE", "SEPAY_INVENTORY_CA_FILE", "SEPAY_INVENTORY_SERVER_NAME"}
	for _, tc := range []struct {
		name   string
		values map[string]string
	}{
		{"profile without endpoint", map[string]string{"SEPAY_INVENTORY_PROFILE": "terminal"}},
		{"credential without endpoint", map[string]string{"SEPAY_INVENTORY_TOKEN_FILE": "secret-file"}},
		{"unknown profile", map[string]string{"SEPAY_INVENTORY_GRPC": "localhost:1", "SEPAY_INVENTORY_PROFILE": "typo"}},
		{"TLS config without profile", map[string]string{"SEPAY_INVENTORY_GRPC": "localhost:1", "SEPAY_INVENTORY_CA_FILE": "ca.pem"}},
		{"missing terminal credential", map[string]string{"SEPAY_INVENTORY_GRPC": "localhost:1", "SEPAY_INVENTORY_PROFILE": "terminal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range keys {
				t.Setenv(key, tc.values[key])
			}
			s := &Service{}
			if err := s.enableInventoryFromEnv(context.Background()); err == nil || s.PaymentInventory != nil {
				t.Fatal("invalid configuration installed an Inventory adapter", err)
			}
		})
	}
}
