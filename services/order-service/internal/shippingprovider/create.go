package shippingprovider

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/internal/persistence"
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

// Tracking discovers a revision number only. The exact historical revision is
// still fetched and checked as authority; creation need not be revision one.
func (a *Adapter) CreationRevision(ctx context.Context, in *shipping.CreateShipmentRequest, id string) (persistence.SourceEvidence, error) {
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c = metadata.AppendToOutgoingContext(c, "authorization", "Bearer "+a.Token)
	current, e := a.Client.TrackShipment(c, &shipping.TrackShipmentRequest{ShipmentId: &id})
	if e != nil {
		return persistence.SourceEvidence{}, domain.E("SHIPPING_CREATION_REVISION_PENDING", 503)
	}
	if current.ShipmentId != id || current.OrderId != in.OrderId || current.PackageId != in.PackageId || current.ReadyGeneration != in.ReadyGeneration || current.SealCode != in.SealCode || current.ResourceVersion < 1 {
		return persistence.SourceEvidence{}, domain.E("SHIPPING_CREATION_MISMATCH", 422)
	}
	proof, e := a.Revision(ctx, id, current.ResourceVersion)
	if e != nil {
		return proof, e
	}
	if proof.Order != in.OrderId || proof.Generation != in.ReadyGeneration || proof.Seal != in.SealCode || proof.Kind != "SHIPMENT_CREATED" {
		return proof, domain.E("SHIPPING_CREATION_REVISION_PENDING", 503)
	}
	return proof, nil
}
