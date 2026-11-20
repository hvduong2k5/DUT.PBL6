package config

import "testing"

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

func TestLoadEnablesShippingV2ForOrderIntegrationByDefault(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_V2_ENABLED", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.V2Enabled {
		t.Fatal("Shipping gRPC v2 must be enabled by default for the Order integration")
	}
	if cfg.PackingV2Enabled || cfg.OrderGRPCEndpoint != "" {
		t.Fatal("packing consumption must remain opt-in for the default Order-owned handoff")
	}
}

func TestLoadRequiresProductionGRPCTLS(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_ENV", "production")
	t.Setenv("SHIPPING_GRPC_TLS_CERT_FILE", "")
	t.Setenv("SHIPPING_GRPC_TLS_KEY_FILE", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected production gRPC TLS to be required")
	}

	t.Setenv("SHIPPING_GRPC_TLS_CERT_FILE", "/run/secrets/shipping.crt")
	cfg, err := Load()
	if err == nil || cfg.GRPCAddress != "" {
		t.Fatal("expected certificate and key to be required as a pair")
	}

	t.Setenv("SHIPPING_GRPC_TLS_KEY_FILE", "/run/secrets/shipping.key")
	if _, err := Load(); err != nil {
		t.Fatalf("expected production TLS configuration to pass validation: %v", err)
	}
}

func TestLoadRequiresServiceTokenAtOrderCompatibleLength(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_INTERNAL_TOKEN", "short")
	if _, err := Load(); err == nil {
		t.Fatal("expected short service token to be rejected")
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
	t.Setenv("SHIPPING_INTERNAL_DELIVERY_ENABLED", "false")
	t.Setenv("SHIPPING_ORDER_GRPC_ENDPOINT", "")
	t.Setenv("SHIPPING_ORDER_INTERNAL_TOKEN", "")
	t.Setenv("SHIPPING_PACKING_V2_ENABLED", "false")
	t.Setenv("SHIPPING_ORDER_CA_FILE", "")
	t.Setenv("SHIPPING_ORDER_SERVER_NAME", "")
	t.Setenv("SHIPPING_ENV", "local")
	t.Setenv("SHIPPING_INTERNAL_TOKEN", "shipping-local-development-secret")
	t.Setenv("SHIPPING_GRPC_TLS_CERT_FILE", "")
	t.Setenv("SHIPPING_GRPC_TLS_KEY_FILE", "")
	t.Setenv("SHIPPING_FAKE_WEBHOOK_SECRET", "fake-secret")
}

func TestInternalDeliveryRequiresExplicitPolicyAndIsOptIn(t *testing.T) {
	setRequiredBaseEnvironment(t)
	t.Setenv("SHIPPING_GHN_ENABLED", "false")
	t.Setenv("SHIPPING_INTERNAL_DELIVERY_ENABLED", "")
	cfg, err := Load()
	if err != nil || cfg.InternalDeliveryEnabled {
		t.Fatalf("internal carrier must be disabled by default: %v", err)
	}
	t.Setenv("SHIPPING_INTERNAL_DELIVERY_ENABLED", "true")
	for _, tc := range []struct{ fee, eta string }{
		{"", "24h"}, {"bad", "24h"}, {"-1", "24h"}, {"1000000001", "24h"},
		{"25000", ""}, {"25000", "invalid"}, {"25000", "0s"}, {"25000", "-1h"}, {"25000", "744h"},
	} {
		t.Run(tc.fee+"/"+tc.eta, func(t *testing.T) {
			t.Setenv("SHIPPING_INTERNAL_DELIVERY_FEE_VND", tc.fee)
			t.Setenv("SHIPPING_INTERNAL_DELIVERY_ETA", tc.eta)
			if _, err := Load(); err == nil {
				t.Fatal("invalid/missing policy accepted")
			}
		})
	}
	t.Setenv("SHIPPING_INTERNAL_DELIVERY_FEE_VND", "25000")
	t.Setenv("SHIPPING_INTERNAL_DELIVERY_ETA", "24h")
	cfg, err = Load()
	if err != nil || !cfg.InternalDeliveryEnabled || cfg.InternalDeliveryFeeVND != 25000 || cfg.InternalDeliveryETA.Hours() != 24 {
		t.Fatalf("valid policy rejected: %v", err)
	}
}
