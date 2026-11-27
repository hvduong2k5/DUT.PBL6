package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	GoodsReceivedEventType   = "vn.omama.procurement.goods.received.v1"
	GoodsReceivedEventTopic  = "procurement.events.v1"
	GoodsReceivedEventSource = "https://omama.vn/services/procurement-service"
)

// GoodsReceivedEvent is a line-scoped CloudEvent. Its ID is the immutable
// goods-receipt line ID, so an outbox retry always republishes the same event.
type GoodsReceivedEvent struct {
	SpecVersion     string                 `json:"specversion"`
	ID              string                 `json:"id"`
	Source          string                 `json:"source"`
	Type            string                 `json:"type"`
	Subject         string                 `json:"subject"`
	Time            time.Time              `json:"time"`
	DataContentType string                 `json:"datacontenttype"`
	Traceparent     string                 `json:"traceparent"`
	Data            GoodsReceivedEventData `json:"data"`
}

type GoodsReceivedEventData struct {
	ReceiptID           string      `json:"receipt_id"`
	ReceiptLineID       string      `json:"receipt_line_id"`
	PurchaseOrderID     string      `json:"purchase_order_id"`
	PurchaseOrderLineID string      `json:"purchase_order_line_id"`
	PONumber            string      `json:"po_number"`
	SupplierID          string      `json:"supplier_id"`
	SupplierName        string      `json:"supplier_name"`
	SKUCode             string      `json:"sku_code"`
	QuantityReceived    json.Number `json:"quantity_received"`
	QuantityAccepted    json.Number `json:"quantity_accepted"`
	QuantityRejected    json.Number `json:"quantity_rejected"`
	Unit                string      `json:"unit"`
	WarehouseID         string      `json:"warehouse_id"`
	BatchCode           string      `json:"batch_code"`
	SupplierLotCode     string      `json:"supplier_lot_code,omitempty"`
	ManufacturedOn      *string     `json:"manufactured_on,omitempty"`
	ExpiresOn           *string     `json:"expires_on,omitempty"`
	QualityStatus       string      `json:"quality_status"`
	ReceivedAt          time.Time   `json:"received_at"`
	InspectedAt         time.Time   `json:"inspected_at"`
}

type OutboxMessage struct {
	ID      string
	Topic   string
	Key     string
	Payload []byte
}

// BuildGoodsReceivedEvents creates one immutable event per inspected receipt
// line. BatchCode is generated deterministically; the supplier lot is carried separately.
func BuildGoodsReceivedEvents(receipt GoodsReceipt) ([]GoodsReceivedEvent, error) {
	if receipt.ID == uuid.Nil || receipt.PurchaseOrderID == uuid.Nil || receipt.SupplierID == uuid.Nil ||
		receipt.Status != GoodsReceiptInspected || receipt.InspectedAt == nil || len(receipt.Lines) == 0 {
		return nil, ErrInvalidState
	}

	events := make([]GoodsReceivedEvent, 0, len(receipt.Lines))
	for _, line := range receipt.Lines {
		if line.ID == uuid.Nil || line.PurchaseOrderLineID == uuid.Nil || strings.TrimSpace(line.MaterialCode) == "" ||
			strings.TrimSpace(line.Unit) == "" || strings.TrimSpace(receipt.WarehouseCode) == "" {
			return nil, ErrInvalidArgument
		}
		received, receivedOK := new(big.Rat).SetString(strings.TrimSpace(line.QuantityReceived))
		accepted, acceptedOK := new(big.Rat).SetString(strings.TrimSpace(line.QuantityAccepted))
		rejected, rejectedOK := new(big.Rat).SetString(strings.TrimSpace(line.QuantityRejected))
		if !receivedOK || !acceptedOK || !rejectedOK || received.Sign() <= 0 || accepted.Sign() < 0 || rejected.Sign() < 0 ||
			new(big.Rat).Add(new(big.Rat).Set(accepted), rejected).Cmp(received) != 0 {
			return nil, ErrInvalidArgument
		}
		unit := strings.ToUpper(strings.TrimSpace(line.Unit))
		if !inventoryReceiptQuantity(received, unit) || len(strings.TrimSpace(line.MaterialCode)) > 50 ||
			strings.TrimSpace(receipt.PurchaseOrderNumber) == "" || len(receipt.PurchaseOrderNumber) > 256 ||
			len(receipt.WarehouseCode) > 256 || len(receipt.SupplierName) > 2000 ||
			receipt.CreatedAt.IsZero() || receipt.InspectedAt.IsZero() {
			return nil, ErrInvalidArgument
		}
		if (line.ManufacturedOn == nil) != (line.ExpiresOn == nil) ||
			(line.ManufacturedOn != nil && !line.ManufacturedOn.Before(*line.ExpiresOn)) {
			return nil, ErrInvalidArgument
		}

		batchCode := fmt.Sprintf("GR-%s-%s", compactUUID(receipt.ID.String()), compactUUID(line.ID.String()))
		events = append(events, GoodsReceivedEvent{
			SpecVersion: "1.0", ID: line.ID.String(), Source: GoodsReceivedEventSource,
			Type: GoodsReceivedEventType, Subject: "sku:" + strings.TrimSpace(line.MaterialCode),
			Time: receipt.InspectedAt.UTC(), DataContentType: "application/json",
			Traceparent: receiptTraceparent(receipt.ID, line.ID),
			Data: GoodsReceivedEventData{
				ReceiptID: receipt.ID.String(), ReceiptLineID: line.ID.String(),
				PurchaseOrderID: receipt.PurchaseOrderID.String(), PurchaseOrderLineID: line.PurchaseOrderLineID.String(),
				PONumber: receipt.PurchaseOrderNumber, SupplierID: receipt.SupplierID.String(), SupplierName: receipt.SupplierName,
				SKUCode: strings.TrimSpace(line.MaterialCode), QuantityReceived: json.Number(decimalString(received)),
				QuantityAccepted: json.Number(decimalString(accepted)), QuantityRejected: json.Number(decimalString(rejected)), Unit: unit,
				WarehouseID: strings.TrimSpace(receipt.WarehouseCode), BatchCode: batchCode,
				SupplierLotCode: strings.TrimSpace(line.SupplierLotCode), ManufacturedOn: dateOnly(line.ManufacturedOn),
				ExpiresOn: dateOnly(line.ExpiresOn), QualityStatus: string(line.QualityStatus),
				ReceivedAt: receipt.CreatedAt.UTC(), InspectedAt: receipt.InspectedAt.UTC(),
			},
		})
	}
	return events, nil
}

// The canonical v1 consumer receives physical stock into quarantine, not sellable
// accepted stock. Match its exact gram/unit conversion without rounding.
func inventoryReceiptQuantity(quantity *big.Rat, unit string) bool {
	if quantity.Cmp(big.NewRat(1, 100)) < 0 {
		return false
	}
	converted := new(big.Rat).Set(quantity)
	switch unit {
	case "KG":
		converted.Mul(converted, big.NewRat(1000, 1))
	case "G", "GRAM", "UNIT", "PIECE":
	default:
		return false
	}
	return converted.IsInt() && converted.Num().IsInt64() && converted.Num().Int64() <= 2147483647
}

// A stable, unsampled receipt-local root, not a claim of an incoming request
// trace. Persisted outbox bytes and rebuilds preserve the same context on retry.
func receiptTraceparent(receiptID, lineID uuid.UUID) string {
	digest := sha256.Sum256([]byte(GoodsReceivedEventSource + ":" + receiptID.String() + ":" + lineID.String()))
	return fmt.Sprintf("00-%x-%x-00", digest[:16], digest[16:24])
}

func compactUUID(value string) string {
	return strings.ReplaceAll(value, "-", "")
}

func decimalString(value *big.Rat) string {
	text := strings.TrimRight(strings.TrimRight(value.FloatString(6), "0"), ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func dateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}
	date := value.Format("2006-01-02")
	return &date
}
