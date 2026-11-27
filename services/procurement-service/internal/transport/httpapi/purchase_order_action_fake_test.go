package httpapi_test

import (
	"context"
	"encoding/json"
	"github.com/omamx/procurement-service/internal/domain"
)

func (r *httpRepository) GetPurchaseOrderActionResult(_ context.Context, actor, key string) (domain.PurchaseOrderActionResult, error) {
	result, ok := r.actionResults[actor+"\x00"+key]
	if !ok {
		return result, domain.ErrNotFound
	}
	return result, nil
}

func (r *httpRepository) SavePurchaseOrderAction(ctx context.Context, order domain.PurchaseOrder, previous domain.PurchaseOrderStatus, cmd domain.PurchaseOrderActionCommand) (domain.PurchaseOrder, error) {
	if _, ok := r.actionResults[cmd.Actor+"\x00"+cmd.Key]; ok {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	result, err := r.SavePurchaseOrder(ctx, order, cmd.ExpectedVersion, previous, cmd.HistoryAction, cmd.Actor, cmd.Reason)
	if err != nil {
		return result, err
	}
	encoded, _ := json.Marshal(result)
	var snapshot domain.PurchaseOrder
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if r.actionResults == nil {
		r.actionResults = make(map[string]domain.PurchaseOrderActionResult)
	}
	r.actionResults[cmd.Actor+"\x00"+cmd.Key] = domain.PurchaseOrderActionResult{RequestHash: cmd.RequestHash, Snapshot: snapshot}
	return result, nil
}
