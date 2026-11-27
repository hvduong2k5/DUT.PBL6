package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"testing"
)

func (r *fakeRepository) GetGoodsReceiptResolutionResult(_ context.Context, actor, key string) (domain.VarianceResolutionResult, error) {
	result, ok := r.resolutionResults[actor+"\x00"+key]
	if !ok {
		return result, domain.ErrNotFound
	}
	return result, nil
}
func (r *fakeRepository) ResolveGoodsReceiptVarianceAction(ctx context.Context, id uuid.UUID, cmd domain.VarianceResolutionCommand) (domain.GoodsReceipt, error) {
	if _, ok := r.resolutionResults[cmd.Actor+"\x00"+cmd.Key]; ok {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	receipt, err := r.GetGoodsReceipt(ctx, id)
	if err != nil {
		return receipt, err
	}
	if receipt.Version != cmd.ExpectedVersion {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	result, err := r.ResolveGoodsReceiptVariance(ctx, id, cmd.VarianceID, cmd.Decision, cmd.Actor, cmd.Note)
	if err != nil {
		return result, err
	}
	encoded, _ := json.Marshal(result)
	var snapshot domain.GoodsReceipt
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if r.resolutionResults == nil {
		r.resolutionResults = map[string]domain.VarianceResolutionResult{}
	}
	r.resolutionResults[cmd.Actor+"\x00"+cmd.Key] = domain.VarianceResolutionResult{RequestHash: cmd.RequestHash, Snapshot: snapshot}
	return result, nil
}

func TestVarianceResolutionReplaysFrozenNormalizedDecision(t *testing.T) {
	repo := newFakeRepository()
	service := application.NewService(repo)
	ctx := context.Background()
	id, varianceID := uuid.New(), uuid.New()
	repo.receipts[id] = domain.GoodsReceipt{ID: id, Status: domain.GoodsReceiptInspected, Version: 2, Variances: []domain.GoodsReceiptVariance{{ID: varianceID, Status: "OPEN"}}}
	first, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, id, varianceID, " accept_variance ", "actor", " Evidence ", "variance-unit-command-0001", 2)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := application.NewService(repo).ResolveGoodsReceiptVarianceWithIdempotency(ctx, id, varianceID, "ACCEPT_VARIANCE", "actor", "Evidence", "variance-unit-command-0001", 2)
	if err != nil || replay.Version != first.Version || replay.Variances[0].ResolutionNote != "Evidence" {
		t.Fatalf("normalized replay: %v", err)
	}
	if _, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, id, varianceID, "REJECT_VARIANCE", "actor", "Evidence", "variance-unit-command-0001", 2); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("changed decision reused key")
	}
	if _, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, id, varianceID, "ACCEPT_VARIANCE", "actor", "Evidence", "variance-fresh-stale-0001", 2); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("stale version accepted")
	}
}
