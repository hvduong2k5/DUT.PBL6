package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
)

type inspectionRaceRepository struct {
	*inspectionWriteSpy
}

func (r *inspectionRaceRepository) InspectGoodsReceipt(ctx context.Context, receipt domain.GoodsReceipt, version int64, actor string) (domain.GoodsReceipt, error) {
	_, err := r.inspectionWriteSpy.InspectGoodsReceipt(ctx, receipt, version, actor)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	// Simulate a concurrent identical commit winning our optimistic write.
	return domain.GoodsReceipt{}, domain.ErrConflict
}

func inspectionFixture() (*inspectionWriteSpy, uuid.UUID, []domain.GoodsReceiptInspectionLine) {
	repo := &inspectionWriteSpy{fakeRepository: newFakeRepository()}
	id, line := uuid.New(), uuid.New()
	repo.receipts[id] = domain.GoodsReceipt{ID: id, PurchaseOrderID: uuid.New(), SupplierID: uuid.New(),
		PurchaseOrderNumber: "PO-1", SupplierName: "Supplier", WarehouseCode: "WH-01",
		CreatedAt: time.Now().UTC(), Status: domain.GoodsReceiptDraft, Version: 1,
		Lines: []domain.GoodsReceiptLine{{ID: line, PurchaseOrderLineID: uuid.New(), MaterialCode: "RICE",
			Unit: "KG", QuantityReceived: "1", QuantityAccepted: "0", QuantityRejected: "0"}},
	}
	return repo, id, []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: line, QuantityAccepted: "1", QuantityRejected: "0"}}
}

func TestInspectionRetryUsesDurableEvidenceAndDoesNotWriteAgain(t *testing.T) {
	repo, id, lines := inspectionFixture()
	first, err := application.NewService(repo).InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines)
	if err != nil {
		t.Fatal(err)
	}
	// A new service instance has no in-memory retry cache.
	replayed, err := application.NewService(repo).InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines)
	if err != nil || replayed.Version != first.Version || repo.writes != 1 || replayed.InspectionRequestHash == "" {
		t.Fatalf("retry rewrote inspection: %v, writes=%d", err, repo.writes)
	}
	for _, tc := range []struct{ actor, key, accepted string }{
		{"inspector", "inspection-key-00001", "0.5"}, {"inspector", "inspection-key-00001", "1.0"},
		{"other", "inspection-key-00001", "1"}, {"inspector", "inspection-key-00002", "1"},
	} {
		changed := append([]domain.GoodsReceiptInspectionLine(nil), lines...)
		changed[0].QuantityAccepted = tc.accepted
		_, err := application.NewService(repo).InspectGoodsReceiptWithIdempotency(context.Background(), id, tc.actor, tc.key, changed)
		if !errors.Is(err, domain.ErrConflict) || repo.writes != 1 {
			t.Fatalf("conflicting replay accepted: %v", err)
		}
	}
}

func TestInspectionRetryCanonicalizesLineOrderAndSurroundingWhitespace(t *testing.T) {
	repo, id, lines := inspectionFixture()
	receipt := repo.receipts[id]
	second := receipt.Lines[0]
	second.ID, second.PurchaseOrderLineID = uuid.New(), uuid.New()
	receipt.Lines = append(receipt.Lines, second)
	repo.receipts[id] = receipt
	lines = append(lines, domain.GoodsReceiptInspectionLine{GoodsReceiptLineID: second.ID, QuantityAccepted: "1", QuantityRejected: "0"})
	service := application.NewService(repo)
	if _, err := service.InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines); err != nil {
		t.Fatal(err)
	}
	lines[0], lines[1] = lines[1], lines[0]
	lines[0].QuantityAccepted = " 1 "
	if _, err := service.InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines); err != nil || repo.writes != 1 {
		t.Fatalf("canonical retry was not replayed: %v", err)
	}
}

func TestConcurrentInspectionCommitCanBeReplayedWithoutRepeatingAction(t *testing.T) {
	repo, id, lines := inspectionFixture()
	race := &inspectionRaceRepository{inspectionWriteSpy: repo}
	result, err := application.NewService(race).InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines)
	if err != nil || result.Status != domain.GoodsReceiptInspected || repo.writes != 1 {
		t.Fatalf("concurrent commit was not recovered: %v", err)
	}
}

func TestInspectionMissingKeyAndInvalidHandoffDoNotConsumeRetryEvidence(t *testing.T) {
	repo, id, lines := inspectionFixture()
	service := application.NewService(repo)
	for _, key := range []string{"", "short", "inspection key with spaces"} {
		if _, err := service.InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", key, lines); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("invalid key accepted: %v", err)
		}
	}
	lines[0].QuantityAccepted = "0.5"
	if _, err := service.InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("invalid inspection accepted: %v", err)
	}
	if repo.writes != 0 || repo.receipts[id].InspectionIdempotencyKey != "" {
		t.Fatal("failed validation consumed key")
	}
	lines[0].QuantityAccepted = "1"
	if _, err := service.InspectGoodsReceiptWithIdempotency(context.Background(), id, "inspector", "inspection-key-00001", lines); err != nil {
		t.Fatal(err)
	}
}
