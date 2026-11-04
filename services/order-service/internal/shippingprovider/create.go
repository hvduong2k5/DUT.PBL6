package shippingprovider

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"google.golang.org/grpc/metadata"
	"time"
)

func (a *Adapter) Create(ctx context.Context, in *shipping.CreateShipmentRequest) (string, error) {
	if a == nil || in == nil || in.PackageId == "" || in.IdempotencyKey == "" {
		return "", domain.E("INVALID_SHIPPING_HANDOFF", 422)
	}
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c = metadata.AppendToOutgoingContext(c, "authorization", "Bearer "+a.Token)
	out, e := a.Client.CreateShipment(c, in)
	if e != nil {
		return "", domain.E("SHIPPING_CREATE_UNAVAILABLE", 503)
	}
	if id, e := uuid.Parse(out.GetShipmentId()); e != nil || id == uuid.Nil {
		return "", domain.E("INVALID_SHIPPING_RESULT", 503)
	}
	return out.ShipmentId, nil
}
