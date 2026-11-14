package promotionprovider

import (
	"context"
	"strings"
	"testing"
)

func TestDialRejectsUntrustedPromotionTransport(t *testing.T) {
	for _, tc := range []struct {
		name, target, token, ca, server string
		local                           bool
	}{
		{name: "short token", target: "localhost:9017", token: "short", local: true},
		{name: "plaintext docker hostname", target: "promotion-service:9017", token: strings.Repeat("p", 32), local: true},
		{name: "plaintext remote address", target: "192.0.2.1:9017", token: strings.Repeat("p", 32), local: true},
		{name: "missing TLS trust", target: "promotion-service:9017", token: strings.Repeat("p", 32)},
		{name: "mixed TLS and plaintext", target: "localhost:9017", token: strings.Repeat("p", 32), ca: "ca.pem", local: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, conn, err := Dial(context.Background(), tc.target, tc.token, tc.ca, tc.server, tc.local); err == nil || conn != nil {
				t.Fatal("untrusted transport was accepted", err)
			}
		})
	}
}
