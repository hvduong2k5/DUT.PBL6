package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func newTestPurchaseOrder(t *testing.T) domain.PurchaseOrder {
	t.Helper()
	delivery := time.Now().UTC().Add(48 * time.Hour)
	order, err := domain.NewPurchaseOrder("PO-TEST-001", uuid.New(), "Supplier", "VND", "test", "creator-1", &delivery, []domain.PurchaseOrderLineInput{
		{MaterialID: uuid.New(), MaterialCode: "SESAME", MaterialName: "Sesame", Quantity: "1.5", Unit: "kg", UnitPriceMinor: 10001},
		{MaterialID: uuid.New(), MaterialCode: "SUGAR", MaterialName: "Sugar", Quantity: "2", Unit: "kg", UnitPriceMinor: 20000},
	})
	if err != nil {
		t.Fatalf("NewPurchaseOrder() error = %v", err)
	}
	return order
}

func TestNewPurchaseOrderCalculatesRoundedLineAndTotal(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if order.Status != domain.PurchaseOrderDraft || order.Version != 1 || order.TotalAmountMinor != 55002 {
		t.Fatalf("unexpected order header: %+v", order)
	}
	if order.Lines[0].LineAmountMinor != 15002 || order.Lines[0].Unit != "KG" {
		t.Fatalf("unexpected line snapshot: %+v", order.Lines[0])
	}
}

func TestNewPurchaseOrderRejectsInvalidLines(t *testing.T) {
	supplierID, materialID := uuid.New(), uuid.New()
	delivery := time.Now().UTC().Add(time.Hour)
	base := []domain.PurchaseOrderLineInput{{MaterialID: materialID, MaterialCode: "M1", MaterialName: "M1", Quantity: "1", Unit: "KG", UnitPriceMinor: 100}}
	tests := []struct {
		name  string
		lines []domain.PurchaseOrderLineInput
	}{
		{name: "zero quantity", lines: []domain.PurchaseOrderLineInput{{MaterialID: materialID, MaterialCode: "M1", MaterialName: "M1", Quantity: "0", Unit: "KG", UnitPriceMinor: 100}}},
		{name: "invalid decimal", lines: []domain.PurchaseOrderLineInput{{MaterialID: materialID, MaterialCode: "M1", MaterialName: "M1", Quantity: "1/2", Unit: "KG", UnitPriceMinor: 100}}},
		{name: "duplicate material", lines: append(base, base[0])},
		{name: "zero price", lines: []domain.PurchaseOrderLineInput{{MaterialID: materialID, MaterialCode: "M1", MaterialName: "M1", Quantity: "1", Unit: "KG", UnitPriceMinor: 0}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := domain.NewPurchaseOrder("PO-TEST", supplierID, "Supplier", "VND", "", "creator", &delivery, tc.lines); err == nil {
				t.Fatal("expected invalid order to be rejected")
			}
		})
	}
	if _, err := domain.NewPurchaseOrder("PO-TEST", supplierID, "Supplier", "VND", "", "creator", nil, nil); err == nil {
		t.Fatal("expected empty lines to be rejected")
	}
}

func TestPurchaseOrderWorkflowAndSeparationOfDuties(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if err := order.Submit("creator-1"); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if order.Status != domain.PurchaseOrderPendingApproval || order.SubmittedBy != "creator-1" {
		t.Fatalf("unexpected submitted state: %+v", order)
	}
	if err := order.Approve("creator-1"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("self-approval error = %v, want ErrConflict", err)
	}
	if err := order.Approve("approver-1"); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if err := order.MarkSent("sender-1"); err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if order.Status != domain.PurchaseOrderSent || order.ApprovedBy != "approver-1" || order.SentBy != "sender-1" {
		t.Fatalf("unexpected sent state: %+v", order)
	}
}

func TestPurchaseOrderRejectAndCancelRequiresReason(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if err := order.Submit("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.Reject("approver-1", ""); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("Reject() without reason error = %v", err)
	}
	if err := order.Reject("approver-1", "price needs review"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderRejected || order.RejectionReason != "price needs review" {
		t.Fatalf("unexpected rejected state: %+v", order)
	}
	if err := order.Cancel("manager-1", "supplier unavailable"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderCancelled {
		t.Fatalf("status = %s, want CANCELLED", order.Status)
	}
	if err := order.Cancel("manager-1", "again"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("repeated cancel error = %v", err)
	}
}

func TestPurchaseOrderPartialAndCompleteReceiptDomainTransition(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if err := order.Submit("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.Approve("approver-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkSent("sender-1"); err != nil {
		t.Fatal(err)
	}
	first := order.Lines[0]
	if err := order.RecordReceived(first.ID, "1"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderPartiallyReceived {
		t.Fatalf("status = %s, want PARTIALLY_RECEIVED", order.Status)
	}
	if err := order.Cancel("manager-1", "cannot cancel after receipt"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("cancel after receipt error = %v", err)
	}
	if err := order.RecordReceived(first.ID, "1"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("over-receive error = %v, want ErrConflict", err)
	}
	if err := order.RecordReceived(first.ID, "0.5"); err != nil {
		t.Fatal(err)
	}
	second := order.Lines[1]
	if err := order.RecordReceived(second.ID, "2"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderReceived {
		t.Fatalf("status = %s, want RECEIVED", order.Status)
	}
}

func TestSupplierResponseRevisionAndSendRetryAreSafe(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if err := order.Submit("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.Approve("approver-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkSent("sender-1"); err != nil {
		t.Fatal(err)
	}
	version, sentAt := order.Version, *order.SentAt
	if err := order.MarkSent("sender-1"); err != nil {
		t.Fatal(err)
	}
	if order.Version != version || !order.SentAt.Equal(sentAt) {
		t.Fatal("retry changed the sent PO")
	}
	if err := order.RecordSupplierResponse(domain.SupplierPOChangeRequested, "recorder-1", "Please adjust delivery quantity"); err != nil {
		t.Fatal(err)
	}
	updated, err := domain.NewPurchaseOrder(order.Number, order.SupplierID, order.SupplierName, order.Currency, "revised terms", order.CreatedBy, order.ExpectedDeliveryDate, []domain.PurchaseOrderLineInput{{MaterialID: uuid.New(), MaterialCode: "SESAME", MaterialName: "Sesame", Quantity: "2", Unit: "KG", UnitPriceMinor: 10001}})
	if err != nil {
		t.Fatal(err)
	}
	if err := order.ReviseAsDraft(updated, "buyer-2", "Accepted supplier change request"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderDraft || order.Revision != 2 || order.CreatedBy != "buyer-2" || order.SupplierRespondedAt != nil || order.Lines[0].Quantity != "2" {
		t.Fatalf("unexpected revised draft: %+v", order)
	}
}

func TestCloseRemainingAfterPartialReceiptPreservesReceivedAmounts(t *testing.T) {
	order := newTestPurchaseOrder(t)
	if err := order.Submit("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.Approve("approver-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkSent("sender-1"); err != nil {
		t.Fatal(err)
	}
	line := order.Lines[0]
	if err := order.RecordReceived(line.ID, "1"); err != nil {
		t.Fatal(err)
	}
	if err := order.CloseRemaining("manager-1", "supplier cannot fulfill balance"); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderClosed || order.Lines[0].ReceivedQuantity != "1.000000" || order.CloseReason == "" {
		t.Fatalf("unexpected closed PO: %+v", order)
	}
	if err := order.RecordReceived(line.ID, "0.5"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("receipt after close error=%v", err)
	}
}

func TestReplaceDraftPreservesIdentityAndRejectsSubmittedOrder(t *testing.T) {
	order := newTestPurchaseOrder(t)
	id, number, createdAt, version := order.ID, order.Number, order.CreatedAt, order.Version
	updated := newTestPurchaseOrder(t)
	if err := order.ReplaceDraft(updated); err != nil {
		t.Fatal(err)
	}
	if order.ID != id || order.Number != number || !order.CreatedAt.Equal(createdAt) || order.Version != version+1 {
		t.Fatalf("draft replacement failed to preserve identity/version: %+v", order)
	}
	if err := order.Submit("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := order.ReplaceDraft(updated); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("editing submitted PO error = %v", err)
	}
}
