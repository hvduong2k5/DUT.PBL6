package main

import "testing"

func TestPeerTransportRequiresExplicitPrivatePlaintext(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, target, flag string
		ok                         bool
	}{
		{"Catalog dev", "TRACEABILITY_CATALOG", "catalog-service:9005", "true", true},
		{"Media dev", "TRACEABILITY_MEDIA", "media-service:9007", "true", true},
		{"loopback", "TRACEABILITY_MEDIA", "127.0.0.1:9007", "true", true},
		{"private IP", "TRACEABILITY_CATALOG", "10.20.30.40:9005", "true", true},
		{"public IP denied", "TRACEABILITY_CATALOG", "8.8.8.8:9005", "true", false},
		{"public DNS denied", "TRACEABILITY_MEDIA", "media.example.com:9007", "true", false},
		{"no auto fallback", "TRACEABILITY_CATALOG", "catalog-service:9005", "", false},
		{"Identity remains TLS", "TRACEABILITY_IDENTITY", "127.0.0.1:9016", "true", false},
		{"Inventory remains TLS", "TRACEABILITY_INVENTORY", "127.0.0.1:9004", "true", false},
		{"invalid port", "TRACEABILITY_MEDIA", "media-service:0", "true", false},
		{"bad flag", "TRACEABILITY_CATALOG", "catalog-service:9005", "yes", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{tc.prefix + "_GRPC_ADDR": tc.target, tc.prefix + "_GRPC_PLAINTEXT": tc.flag}
			target, transport, err := peerTransport(func(k string) string { return values[k] }, tc.prefix)
			if (err == nil) != tc.ok {
				t.Fatal("incorrect transport policy", err)
			}
			if tc.ok && (target != tc.target || transport == nil || transport.Info().SecurityProtocol != "insecure") {
				t.Fatal("unexpected peer transport")
			}
		})
	}
}
