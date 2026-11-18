package config

import "testing"

func TestLoadLeavesGHNDisabledWithoutCredentials(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_GHN_ENABLED", "false")
	t.Setenv("SHIPPING_GHN_TOKEN", "")
	t.Setenv("SHIPPING_GHN_SHOP_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GHNEnabled {
		t.Fatal("GHN must remain disabled by default")
	}
}

func TestLoadRequiresCompleteGHNCredentialsWhenEnabled(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_GHN_ENABLED", "true")
	t.Setenv("SHIPPING_GHN_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected incomplete GHN credentials to be rejected")
	}

	t.Setenv("SHIPPING_GHN_TOKEN", "staging-token")
	t.Setenv("SHIPPING_GHN_SHOP_ID", "92837")
	t.Setenv("SHIPPING_GHN_FROM_DISTRICT_ID", "1452")
	t.Setenv("SHIPPING_GHN_FROM_WARD_CODE", "21007")
	t.Setenv("SHIPPING_GHN_WEBHOOK_SECRET", "random-webhook-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.GHNEnabled || cfg.GHNShopID != 92837 || cfg.GHNFromDistrictID != 1452 {
		t.Fatalf("unexpected GHN configuration: %+v", cfg)
	}
}

func setRequiredBaseEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("SHIPPING_INTERNAL_TOKEN", "internal-token")
	t.Setenv("SHIPPING_FAKE_WEBHOOK_SECRET", "fake-secret")
	t.Setenv("SHIPPING_ORDER_GRPC_ENDPOINT", "")
	t.Setenv("SHIPPING_ORDER_INTERNAL_TOKEN", "")
}

func TestPackingV2RequiresOrderTLS(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_GHN_ENABLED", "false")
	t.Setenv("SHIPPING_PACKING_V2_ENABLED", "true")
	t.Setenv("SHIPPING_ORDER_GRPC_ENDPOINT", "order.internal:9004")
	t.Setenv("SHIPPING_ORDER_INTERNAL_TOKEN", "fixture-token")
	t.Setenv("SHIPPING_ORDER_CA_FILE", "")
	t.Setenv("SHIPPING_ORDER_SERVER_NAME", "")
	if _, err := Load(); err == nil {
		t.Fatal("packing v2 accepted a plaintext Order connection")
	}
	t.Setenv("SHIPPING_ORDER_CA_FILE", "/run/secrets/order-ca.pem")
	t.Setenv("SHIPPING_ORDER_SERVER_NAME", "order.internal")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}
