package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func TestBuildGoodsReceivedEventsCreatesStableLineEventWithAcceptedQuantity(t *testing.T) {
	receiptID := uuid.MustParse("6a9eb64a-3e83-4df1-a27e-a6da040a9612")
	lineID := uuid.MustParse("6835ff40-11da-4858-8d50-a6a86471e087")
	poLineID := uuid.MustParse("2485b858-c092-4175-b547-29330d7806e0")
	inspectedAt := time.Date(2026, 11, 13, 9, 30, 0, 0, time.FixedZone("ICT", 7*60*60))
	manufacturedOn := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	expiresOn := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	receipt := domain.GoodsReceipt{
		ID: receiptID, Number: "GR-20261113-A1B2C3D4", PurchaseOrderID: uuid.New(),
		PurchaseOrderNumber: "PO-20261110-01020304", SupplierID: uuid.New(), SupplierName: "Supplier",
		WarehouseCode: "WH-HUE", Status: domain.GoodsReceiptInspected,
		CreatedAt: inspectedAt.Add(-time.Hour), InspectedAt: &inspectedAt,
		Lines: []domain.GoodsReceiptLine{{
			ID: lineID, PurchaseOrderLineID: poLineID, MaterialCode: "SESAME-RAW", Unit: "KG",
			QuantityReceived: "2", QuantityAccepted: "1.5", QuantityRejected: "0.5",
			QualityStatus: domain.GoodsReceiptQualityPartial, SupplierLotCode: "SUP-LOT-42", ManufacturedOn: &manufacturedOn, ExpiresOn: &expiresOn,
		}},
	}

	events, err := domain.BuildGoodsReceivedEvents(receipt)
	if err != nil {
		t.Fatalf("BuildGoodsReceivedEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	if event.ID != lineID.String() || event.Type != domain.GoodsReceivedEventType || event.Subject != "sku:SESAME-RAW" {
		t.Fatalf("unexpected CloudEvent identity: %+v", event)
	}
	if event.Data.QuantityReceived.String() != "2" || event.Data.QuantityAccepted.String() != "1.5" || event.Data.QuantityRejected.String() != "0.5" {
		t.Fatalf("event quantities = received %q, accepted %q, rejected %q", event.Data.QuantityReceived, event.Data.QuantityAccepted, event.Data.QuantityRejected)
	}
	wantBatchCode := "GR-6a9eb64a3e834df1a27ea6da040a9612-6835ff4011da48588d50a6a86471e087"
	if event.Data.BatchCode != wantBatchCode || event.Data.SupplierLotCode != "SUP-LOT-42" || event.Data.WarehouseID != "WH-HUE" {
		t.Fatalf("unexpected inventory handoff fields: %+v", event.Data)
	}
	if event.Data.ManufacturedOn == nil || *event.Data.ManufacturedOn != "2026-11-01" || event.Data.ExpiresOn == nil || *event.Data.ExpiresOn != "2027-03-01" {
		t.Fatalf("lot dates must use date-only values: manufactured=%v expires=%v", event.Data.ManufacturedOn, event.Data.ExpiresOn)
	}
	if !event.Time.Equal(inspectedAt.UTC()) || !event.Data.InspectedAt.Equal(inspectedAt.UTC()) {
		t.Fatalf("event time was not normalized to UTC: event=%s inspected=%s", event.Time, event.Data.InspectedAt)
	}
	first, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	secondEvents, err := domain.BuildGoodsReceivedEvents(receipt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(secondEvents[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("same inspected receipt produced different event bytes:\n%s\n%s", first, second)
	}
}

func TestBuildGoodsReceivedEventsUsesDeterministicBatchCodeWithoutSupplierLot(t *testing.T) {
	receipt := inspectedReceiptFixture()
	receipt.Lines[0].SupplierLotCode = "  "
	events, err := domain.BuildGoodsReceivedEvents(receipt)
	if err != nil {
		t.Fatalf("BuildGoodsReceivedEvents() error = %v", err)
	}
	want := "GR-" + strings.ReplaceAll(receipt.ID.String(), "-", "") + "-" + strings.ReplaceAll(receipt.Lines[0].ID.String(), "-", "")
	if events[0].Data.BatchCode != want {
		t.Fatalf("batch code = %q, want %q", events[0].Data.BatchCode, want)
	}
}

func TestBuildGoodsReceivedEventsRejectsUninspectedOrInconsistentReceipt(t *testing.T) {
	receipt := inspectedReceiptFixture()
	receipt.Status = domain.GoodsReceiptDraft
	if _, err := domain.BuildGoodsReceivedEvents(receipt); err == nil {
		t.Fatal("expected draft receipt to be rejected")
	}

	receipt = inspectedReceiptFixture()
	receipt.Lines[0].QuantityAccepted = "1"
	if _, err := domain.BuildGoodsReceivedEvents(receipt); err == nil {
		t.Fatal("expected inconsistent inspected quantities to be rejected")
	}
}

func inspectedReceiptFixture() domain.GoodsReceipt {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	return domain.GoodsReceipt{
		ID:                  uuid.MustParse("6a9eb64a-3e83-4df1-a27e-a6da040a9612"),
		PurchaseOrderID:     uuid.MustParse("7a9eb64a-3e83-4df1-a27e-a6da040a9612"),
		SupplierID:          uuid.MustParse("8a9eb64a-3e83-4df1-a27e-a6da040a9612"),
		PurchaseOrderNumber: "PO-1", SupplierName: "Supplier", WarehouseCode: "WH-1",
		Status: domain.GoodsReceiptInspected, CreatedAt: now.Add(-time.Hour), InspectedAt: &now,
		Lines: []domain.GoodsReceiptLine{{
			ID:                  uuid.MustParse("6835ff40-11da-4858-8d50-a6a86471e087"),
			PurchaseOrderLineID: uuid.MustParse("2485b858-c092-4175-b547-29330d7806e0"),
			MaterialCode:        "MAT-1", Unit: "KG", QuantityReceived: "2", QuantityAccepted: "2",
			QuantityRejected: "0", QualityStatus: domain.GoodsReceiptQualityAccepted,
		}},
	}
}
