package shippingprovider

import (
	"context"
	"encoding/json"
	"errors"
	api "github.com/omamx/order-service/internal/gen/shipping/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

type revisionPeer struct {
	api.ShippingServiceClient
	revision *api.ShipmentRevision
	err      error
}

func (p revisionPeer) GetShipmentRevision(context.Context, *api.GetShipmentRevisionRequest, ...grpc.CallOption) (*api.ShipmentRevision, error) {
	return p.revision, p.err
}

func TestNotificationRequiresExactProviderHistory(t *testing.T) {
	const shipment = "11111111-1111-4111-8111-111111111111"
	const order = "22222222-2222-4222-8222-222222222222"
	at := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	proof := &api.ShipmentRevision{ShipmentId: shipment, OrderId: order, PackageId: "package", ResourceVersion: 4, ReadyGeneration: 2, Kind: "SHIPMENT_DELIVERED", SealCode: "provider-seal", OccurredAt: timestamppb.New(at)}
	notice := func(change func(map[string]any)) []byte {
		n := map[string]any{"specversion": "1.0", "id": "33333333-3333-4333-8333-333333333333", "source": "https://omama.vn/services/shipping-service", "type": "vn.omama.shipping.shipment.delivered.v2", "subject": "shipment_id:" + shipment, "time": at.Add(321 * time.Nanosecond), "data": map[string]any{"shipment_id": shipment, "order_id": order, "resource_version": 4, "ready_generation": 2}}
		if change != nil {
			change(n)
		}
		raw, _ := json.Marshal(n)
		return raw
	}
	a := &Adapter{Client: revisionPeer{revision: proof}, Token: "test-token-012345678901234567890123"}
	event, err := a.Notification(context.Background(), notice(nil))
	if err != nil || event.Seal != "provider-seal" || event.OrderID != order || event.Version != 4 {
		t.Fatal("valid provider history", event, err)
	}
	cases := map[string]func(map[string]any){
		"forged source": func(n map[string]any) { n["source"] = "attacker" },
		"forged order": func(n map[string]any) {
			n["data"].(map[string]any)["order_id"] = "44444444-4444-4444-8444-444444444444"
		},
		"forged generation":   func(n map[string]any) { n["data"].(map[string]any)["ready_generation"] = 3 },
		"wrong revision":      func(n map[string]any) { n["data"].(map[string]any)["resource_version"] = 5 },
		"rewritten timestamp": func(n map[string]any) { n["time"] = at.Add(time.Second) },
		"kind mismatch":       func(n map[string]any) { n["type"] = "vn.omama.shipping.shipment.created.v2" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := a.Notification(context.Background(), notice(change)); err == nil {
				t.Fatal("notification bypassed provider revision")
			}
		})
	}
	a.Client = revisionPeer{err: errors.New("provider unavailable")}
	if _, err := a.Notification(context.Background(), notice(nil)); err == nil {
		t.Fatal("unavailable history treated as delivered")
	}
}

func TestShippingConfigurationCannotDowngradeTLS(t *testing.T) {
	for _, target := range []string{"shipping:9012", "192.0.2.1:9012", "localhost:9012"} {
		_, conn, err := Dial(context.Background(), target, "test-token-012345678901234567890123", "", "", target != "localhost:9012")
		if conn != nil {
			conn.Close()
		}
		if err == nil {
			t.Fatal("unverified transport", target)
		}
	}
}
