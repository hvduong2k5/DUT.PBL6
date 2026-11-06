package domain

import (
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GoodsReceiptStatus string

const (
	GoodsReceiptDraft     GoodsReceiptStatus = "DRAFT"
	GoodsReceiptInspected GoodsReceiptStatus = "INSPECTED"
)

type GoodsReceiptQualityStatus string

const (
	GoodsReceiptQualityPending  GoodsReceiptQualityStatus = "PENDING"
	GoodsReceiptQualityAccepted GoodsReceiptQualityStatus = "ACCEPTED"
	GoodsReceiptQualityPartial  GoodsReceiptQualityStatus = "PARTIALLY_ACCEPTED"
	GoodsReceiptQualityRejected GoodsReceiptQualityStatus = "REJECTED"
)

type GoodsReceiptLine struct {
	ID                  uuid.UUID                 `json:"id"`
	PurchaseOrderLineID uuid.UUID                 `json:"purchase_order_line_id"`
	MaterialID          uuid.UUID                 `json:"material_id"`
	MaterialCode        string                    `json:"material_code"`
	MaterialName        string                    `json:"material_name"`
	Unit                string                    `json:"unit"`
	QuantityReceived    string                    `json:"quantity_received"`
	QuantityAccepted    string                    `json:"quantity_accepted"`
	QuantityRejected    string                    `json:"quantity_rejected"`
	QualityStatus       GoodsReceiptQualityStatus `json:"quality_status"`
	SupplierLotCode     string                    `json:"supplier_lot_code,omitempty"`
	ManufacturedOn      *time.Time                `json:"manufactured_on,omitempty"`
	ExpiresOn           *time.Time                `json:"expires_on,omitempty"`
	HumidityPercent     string                    `json:"humidity_percent,omitempty"`
	InspectionNotes     string                    `json:"inspection_notes,omitempty"`
}

type GoodsReceipt struct {
	ID                  uuid.UUID          `json:"id"`
	Number              string             `json:"number"`
	PurchaseOrderID     uuid.UUID          `json:"purchase_order_id"`
	PurchaseOrderNumber string             `json:"purchase_order_number"`
	SupplierID          uuid.UUID          `json:"supplier_id"`
	SupplierName        string             `json:"supplier_name"`
	WarehouseCode       string             `json:"warehouse_code"`
	Status              GoodsReceiptStatus `json:"status"`
	Lines               []GoodsReceiptLine `json:"lines"`
	ReceivedBy          string             `json:"received_by"`
	InspectedBy         string             `json:"inspected_by,omitempty"`
	InspectedAt         *time.Time         `json:"inspected_at,omitempty"`
	Notes               string             `json:"notes,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	Version             int64              `json:"version"`
}

type GoodsReceiptLineInput struct {
	PurchaseOrderLineID uuid.UUID
	MaterialID          uuid.UUID
	MaterialCode        string
	MaterialName        string
	Unit                string
	QuantityReceived    string
	SupplierLotCode     string
	ManufacturedOn      *time.Time
	ExpiresOn           *time.Time
}

type GoodsReceiptInspectionLine struct {
	GoodsReceiptLineID uuid.UUID
	QuantityAccepted   string
	QuantityRejected   string
	HumidityPercent    string
	InspectionNotes    string
}

func NewGoodsReceipt(number string, purchaseOrder PurchaseOrder, warehouseCode, receivedBy, notes string, lines []GoodsReceiptLineInput) (GoodsReceipt, error) {
	number = strings.TrimSpace(number)
	warehouseCode = strings.ToUpper(strings.TrimSpace(warehouseCode))
	receivedBy = strings.TrimSpace(receivedBy)
	if number == "" || purchaseOrder.ID == uuid.Nil || purchaseOrder.SupplierID == uuid.Nil || purchaseOrder.Status != PurchaseOrderSent && purchaseOrder.Status != PurchaseOrderPartiallyReceived || warehouseCode == "" || receivedBy == "" || len(lines) == 0 {
		return GoodsReceipt{}, ErrInvalidArgument
	}
	poLines := make(map[uuid.UUID]PurchaseOrderLine, len(purchaseOrder.Lines))
	for _, line := range purchaseOrder.Lines {
		poLines[line.ID] = line
	}
	seen := make(map[uuid.UUID]struct{}, len(lines))
	result := GoodsReceipt{
		ID: uuid.New(), Number: number, PurchaseOrderID: purchaseOrder.ID,
		PurchaseOrderNumber: purchaseOrder.Number, SupplierID: purchaseOrder.SupplierID,
		SupplierName: purchaseOrder.SupplierName, WarehouseCode: warehouseCode,
		Status: GoodsReceiptDraft, ReceivedBy: receivedBy, Notes: strings.TrimSpace(notes),
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Version: 1,
		Lines: make([]GoodsReceiptLine, 0, len(lines)),
	}
	for _, input := range lines {
		poLine, exists := poLines[input.PurchaseOrderLineID]
		if !exists || poLine.MaterialID != input.MaterialID || input.QuantityReceived == "" {
			return GoodsReceipt{}, ErrInvalidArgument
		}
		if _, duplicate := seen[input.PurchaseOrderLineID]; duplicate {
			return GoodsReceipt{}, ErrInvalidArgument
		}
		seen[input.PurchaseOrderLineID] = struct{}{}
		quantityText := strings.TrimSpace(input.QuantityReceived)
		quantity, ok := new(big.Rat).SetString(quantityText)
		if !ok || !purchaseQuantityPattern.MatchString(quantityText) || quantity.Sign() <= 0 {
			return GoodsReceipt{}, ErrInvalidArgument
		}
		manufacturedOn, expiresOn, err := validateLotDates(input.ManufacturedOn, input.ExpiresOn)
		if err != nil {
			return GoodsReceipt{}, err
		}
		result.Lines = append(result.Lines, GoodsReceiptLine{
			ID: uuid.New(), PurchaseOrderLineID: poLine.ID, MaterialID: poLine.MaterialID,
			MaterialCode: poLine.MaterialCode, MaterialName: poLine.MaterialName, Unit: poLine.Unit,
			QuantityReceived: quantityText, QuantityAccepted: "0", QuantityRejected: "0",
			QualityStatus: GoodsReceiptQualityPending, SupplierLotCode: strings.TrimSpace(input.SupplierLotCode),
			ManufacturedOn: manufacturedOn, ExpiresOn: expiresOn,
		})
	}
	return result, nil
}

func validateLotDates(manufacturedOn, expiresOn *time.Time) (*time.Time, *time.Time, error) {
	if (manufacturedOn == nil) != (expiresOn == nil) {
		return nil, nil, ErrInvalidArgument
	}
	if manufacturedOn == nil {
		return nil, nil, nil
	}
	manufactured := time.Date(manufacturedOn.Year(), manufacturedOn.Month(), manufacturedOn.Day(), 0, 0, 0, 0, time.UTC)
	expires := time.Date(expiresOn.Year(), expiresOn.Month(), expiresOn.Day(), 0, 0, 0, 0, time.UTC)
	if !manufactured.Before(expires) {
		return nil, nil, ErrInvalidArgument
	}
	return &manufactured, &expires, nil
}

func (r *GoodsReceipt) Inspect(actor string, inspections []GoodsReceiptInspectionLine) error {
	actor = strings.TrimSpace(actor)
	if r.Status != GoodsReceiptDraft || actor == "" || len(inspections) != len(r.Lines) {
		return ErrInvalidState
	}
	byLine := make(map[uuid.UUID]GoodsReceiptInspectionLine, len(inspections))
	for _, inspection := range inspections {
		if _, exists := byLine[inspection.GoodsReceiptLineID]; exists {
			return ErrInvalidArgument
		}
		byLine[inspection.GoodsReceiptLineID] = inspection
	}
	type validatedInspection struct {
		accepted, rejected, humidity, notes string
		status                              GoodsReceiptQualityStatus
	}
	validated := make(map[uuid.UUID]validatedInspection, len(inspections))
	for i := range r.Lines {
		line := &r.Lines[i]
		inspection, exists := byLine[line.ID]
		if !exists {
			return ErrInvalidArgument
		}
		acceptedText, rejectedText := strings.TrimSpace(inspection.QuantityAccepted), strings.TrimSpace(inspection.QuantityRejected)
		accepted, acceptedOK := new(big.Rat).SetString(acceptedText)
		rejected, rejectedOK := new(big.Rat).SetString(rejectedText)
		received, _ := new(big.Rat).SetString(line.QuantityReceived)
		if !acceptedOK || !rejectedOK || !purchaseQuantityPattern.MatchString(acceptedText) || !purchaseQuantityPattern.MatchString(rejectedText) || accepted.Sign() < 0 || rejected.Sign() < 0 {
			return ErrInvalidArgument
		}
		if new(big.Rat).Add(new(big.Rat).Set(accepted), rejected).Cmp(received) != 0 {
			return ErrInvalidArgument
		}
		humidity := strings.TrimSpace(inspection.HumidityPercent)
		if humidity != "" {
			humidityValue, ok := new(big.Rat).SetString(humidity)
			maxHumidity := big.NewRat(100, 1)
			if !ok || !purchaseQuantityPattern.MatchString(humidity) || humidityValue.Sign() < 0 || humidityValue.Cmp(maxHumidity) > 0 {
				return ErrInvalidArgument
			}
		}
		result := validatedInspection{accepted: acceptedText, rejected: rejectedText, humidity: humidity, notes: strings.TrimSpace(inspection.InspectionNotes)}
		switch {
		case accepted.Sign() == 0:
			result.status = GoodsReceiptQualityRejected
		case rejected.Sign() == 0:
			result.status = GoodsReceiptQualityAccepted
		default:
			result.status = GoodsReceiptQualityPartial
		}
		validated[line.ID] = result
	}
	for i := range r.Lines {
		line := &r.Lines[i]
		result := validated[line.ID]
		line.QuantityAccepted = result.accepted
		line.QuantityRejected = result.rejected
		line.HumidityPercent = result.humidity
		line.InspectionNotes = result.notes
		line.QualityStatus = result.status
	}
	r.Status = GoodsReceiptInspected
	r.InspectedBy = actor
	now := time.Now().UTC()
	r.InspectedAt = &now
	r.touch()
	return nil
}

func (r *GoodsReceipt) touch() {
	r.UpdatedAt = time.Now().UTC()
	r.Version++
}
