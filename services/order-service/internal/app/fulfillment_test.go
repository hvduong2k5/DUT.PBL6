package app

import (
	"context"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/fulfillmentprovider"
	"strings"
	"testing"
)

func TestRealPackingCannotUseLegacyStopOrCompletionAuthority(t *testing.T) {
	s := &Service{Fulfillment: &fulfillmentprovider.Adapter{}}
	for _, active := range []bool{true, false} {
		e := s.setCoreHold(context.Background(), domain.Principal{Role: "ADMIN"}, "order", 1, active, "key")
		if e == nil || e.Error() != "FULFILLMENT_STOP_CAPABILITY_UNAVAILABLE" {
			t.Fatal("legacy barrier reached for real packing", e)
		}
	}
	t.Setenv("ORDER_FULFILLMENT_GRPC", "127.0.0.1:9002")
	s.C.AutoComplete = true
	if e := s.EnableFulfillmentFromEnv(context.Background()); e == nil || !strings.Contains(e.Error(), "ENABLE_AUTO_COMPLETE=false") {
		t.Fatal("default completion authority enabled for real packing", e)
	}
}
