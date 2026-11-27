package httpapi_test

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func (r *httpRepository) GetGoodsReceiptResolutionResult(_ context.Context, actor, key string) (domain.VarianceResolutionResult, error) {
	result, ok := r.resolutionResults[actor+"\x00"+key]
	if !ok {
		return result, domain.ErrNotFound
	}
	return result, nil
}
func (r *httpRepository) ResolveGoodsReceiptVarianceAction(ctx context.Context, id uuid.UUID, cmd domain.VarianceResolutionCommand) (domain.GoodsReceipt, error) {
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
