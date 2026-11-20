package domain

import (
	"github.com/google/uuid"
	"time"
)

type DriverShipmentSummary struct {
	ShipmentID        uuid.UUID      `json:"shipment_id"`
	ShipmentNumber    string         `json:"shipment_number"`
	OrderID           string         `json:"order_id"`
	PackageID         string         `json:"package_id"`
	Status            ShipmentStatus `json:"status"`
	Revision          int64          `json:"revision"`
	AssignmentVersion int64          `json:"assignment_version"`
	Accepted          bool           `json:"accepted"`
}

type DriverShipmentList struct {
	Items  []DriverShipmentSummary `json:"items"`
	Total  int64                   `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

type DriverShipmentDetail struct {
	DriverShipmentSummary
	Recipient           Recipient  `json:"recipient"`
	Parcel              Parcel     `json:"parcel"`
	TrackingCode        string     `json:"tracking_code"`
	HandedOverAt        *time.Time `json:"handed_over_at"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at"`
}
