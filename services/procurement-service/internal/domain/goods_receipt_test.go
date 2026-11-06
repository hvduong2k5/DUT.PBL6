package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/omamx/procurement-service/internal/domain"
)

func receiptFixture(t *testing.T) (domain.PurchaseOrder, domain.GoodsReceipt) {
	t.Helper()
	supplier, err := domain.NewSupplier("SUP-GR", "Supplier", "COMPANY", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := domain.NewRawMaterial("RICE", "Rice", "KG", "")
	if err != nil {
		t.Fatal(err)
	}
	po, err := domain.NewPurchaseOrder("PO-GR", supplier.ID, supplier.Name, "VND", "", "buyer", nil, []domain.PurchaseOrderLineInput{{MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name, Quantity: "10", Unit: "KG", UnitPriceMinor: 100}})
	if err != nil {
		t.Fatal(err)
	}
	po.Status = domain.PurchaseOrderSent
	receipt, err := domain.NewGoodsReceipt("GR-1", po, "WH-01", "receiver", "", []domain.GoodsReceiptLineInput{{PurchaseOrderLineID: po.Lines[0].ID, MaterialID: material.ID, QuantityReceived: "4"}})
	if err != nil {
		t.Fatal(err)
	}
	return po, receipt
}

func TestGoodsReceiptInspectionCapturesAcceptedAndRejectedQuantities(t *testing.T) {
	po, receipt := receiptFixture(t)
	if err := receipt.Inspect("inspector", []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: receipt.Lines[0].ID, QuantityAccepted: "3.5", QuantityRejected: "0.5", HumidityPercent: "12.4", InspectionNotes: "one bag damaged"}}); err != nil {
		t.Fatal(err)
	}
	line := receipt.Lines[0]
	if receipt.Status != domain.GoodsReceiptInspected || line.QualityStatus != domain.GoodsReceiptQualityPartial || line.QuantityAccepted != "3.5" || line.QuantityRejected != "0.5" || receipt.InspectedAt == nil {
		t.Fatalf("unexpected inspected receipt: %+v", receipt)
	}
	if err := po.RecordReceiptQuantities(po.Lines[0].ID, line.QuantityAccepted, line.QuantityRejected); err != nil {
		t.Fatal(err)
	}
	if po.Lines[0].AcceptedQuantity != "3.500000" || po.Lines[0].ReceivedQuantity != "4.000000" || po.Status != domain.PurchaseOrderPartiallyReceived {
		t.Fatalf("PO totals/status did not reflect inspection: %+v", po)
	}
}

func TestGoodsReceiptRejectsMismatchedInspectionAndInvalidLotDates(t *testing.T) {
	_, receipt := receiptFixture(t)
	if err := receipt.Inspect("inspector", []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: receipt.Lines[0].ID, QuantityAccepted: "2", QuantityRejected: "1"}}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("mismatched quantities error=%v", err)
	}
	_, receipt = receiptFixture(t)
	_, err := domain.NewGoodsReceipt("GR-2", func() domain.PurchaseOrder { p, _ := receiptFixture(t); return p }(), "WH", "receiver", "", []domain.GoodsReceiptLineInput{{PurchaseOrderLineID: receipt.Lines[0].PurchaseOrderLineID, MaterialID: receipt.Lines[0].MaterialID, QuantityReceived: "1", ManufacturedOn: timePtr(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)), ExpiresOn: timePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))}})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("invalid lot dates error=%v", err)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
