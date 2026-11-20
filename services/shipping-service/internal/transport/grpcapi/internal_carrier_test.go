package grpcapi

import (
	"github.com/omamx/shipping-service/internal/carrier"
	"testing"
)

func TestInternalTrackingPrefixDoesNotFallBackToFake(t *testing.T) {
	if got := carrierCodeFromTracking("INT-a0000000-0000-0000-0000-000000000001"); got != carrier.InternalCode {
		t.Fatalf("internal tracking routed to %s", got)
	}
	if got := carrierCodeFromTracking("FAKE123"); got != carrier.FakeCode {
		t.Fatalf("fake carrier changed: %s", got)
	}
}
