package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	RevisionKindCreated    = "SHIPMENT_CREATED"
	RevisionKindDispatched = "SHIPMENT_DISPATCHED"
	RevisionKindDelivered  = "SHIPMENT_DELIVERED"
	RevisionKindFailed     = "SHIPMENT_FAILED"
)

var ErrShipmentRevisionNotFound = errors.New("shipment revision not found")

type ShipmentRevision struct {
	ShipmentID         uuid.UUID
	OrderID            string
	PackageID          string
	ResourceVersion    int64
	ReadyGeneration    int64
	Kind               string
	SealCode           string
	OccurredAt         time.Time
	SettlementVerified bool
}

func NewShipmentRevision(shipment *Shipment, event DomainEvent) (*ShipmentRevision, bool) {
	kind := revisionKindForEvent(event.Type)
	if kind == "" || shipment.ReadyGeneration < 1 || shipment.SealCode == "" {
		return nil, false
	}
	return &ShipmentRevision{
		ShipmentID: shipment.ID, OrderID: shipment.OrderID, PackageID: shipment.PackageID,
		ResourceVersion: shipment.Revision, ReadyGeneration: shipment.ReadyGeneration,
		Kind: kind, SealCode: shipment.SealCode, OccurredAt: event.OccurredAt.UTC(),
		SettlementVerified: false,
	}, true
}

func revisionKindForEvent(eventType string) string {
	switch eventType {
	case "vn.omama.shipping.shipment.created.v1", "vn.omama.shipping.shipment.created.v2":
		return RevisionKindCreated
	case "vn.omama.shipping.shipment.handed-over.v1", "vn.omama.shipping.shipment.handed-over.v2":
		return RevisionKindDispatched
	case "vn.omama.shipping.shipment.delivered.v1", "vn.omama.shipping.shipment.delivered.v2":
		return RevisionKindDelivered
	case "vn.omama.shipping.shipment.delivery-failed.v1", "vn.omama.shipping.shipment.delivery-failed.v2":
		return RevisionKindFailed
	default:
		return ""
	}
}
