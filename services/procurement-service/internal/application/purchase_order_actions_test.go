package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
)

func (r *fakeRepository) GetPurchaseOrderActionResult(_ context.Context, actor, key string) (domain.PurchaseOrderActionResult, error) {
	result, ok := r.actionResults[actor+"\x00"+key]
	if !ok {
		return result, domain.ErrNotFound
	}
	return result, nil
}

func (r *fakeRepository) SavePurchaseOrderAction(ctx context.Context, order domain.PurchaseOrder, previous domain.PurchaseOrderStatus, cmd domain.PurchaseOrderActionCommand) (domain.PurchaseOrder, error) {
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

type lateActionLookup struct {
	*fakeRepository
	hidden bool
}

func (r *lateActionLookup) GetPurchaseOrderActionResult(ctx context.Context, actor, key string) (domain.PurchaseOrderActionResult, error) {
	if !r.hidden {
		r.hidden = true
		return domain.PurchaseOrderActionResult{}, domain.ErrNotFound
	}
	return r.fakeRepository.GetPurchaseOrderActionResult(ctx, actor, key)
}

func TestPOActionReplaysOriginalSnapshotAfterLifecycleAdvances(t *testing.T) {
	service, repo, supplier, material := makeApplicationFixture(t)
	ctx := context.Background()
	po, err := service.CreatePurchaseOrder(ctx, application.PurchaseOrderRequest{SupplierID: supplier, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material, Quantity: "10"}}}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "creator", "po-submit-test-0001", "SUBMIT", po.Version, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "creator", "po-self-approve-0001", "APPROVE", submitted.Version, ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("self approval accepted: %v", err)
	}
	approved, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "checker", "po-approve-test-0001", "APPROVE", submitted.Version, "")
	if err != nil {
		t.Fatal(err)
	}
	sent, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "sender", "po-send-test-0001", "SEND", approved.Version, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, replayService := range []*application.Service{application.NewService(repo), application.NewService(&lateActionLookup{fakeRepository: repo})} {
		result, err := replayService.ExecutePurchaseOrderAction(ctx, po.ID, "creator", "po-submit-test-0001", "SUBMIT", po.Version, "")
		if err != nil || result.Status != domain.PurchaseOrderPendingApproval || result.Version != submitted.Version || repo.orders[po.ID].Version != sent.Version {
			t.Fatalf("replay changed PO or snapshot: %v", err)
		}
	}
	for _, tc := range []struct {
		key, action string
		version     int64
		reason      string
	}{
		{"po-submit-test-0001", "APPROVE", po.Version, ""},
		{"po-submit-test-0001", "SUBMIT", submitted.Version, ""},
		{"po-new-stale-000001", "SUBMIT", po.Version, ""},
		{"po-new-send-0000001", "SEND", sent.Version, ""},
	} {
		_, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "creator", tc.key, tc.action, tc.version, tc.reason)
		if !errors.Is(err, domain.ErrConflict) && !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("invalid command accepted: %v", err)
		}
	}
	if len(repo.actionResults) != 3 {
		t.Fatalf("failed commands consumed keys: %d", len(repo.actionResults))
	}
}

func TestPORejectRequiresReasonAndReplaysNormalizedEvidence(t *testing.T) {
	service, repo, supplier, material := makeApplicationFixture(t)
	ctx := context.Background()
	po, err := service.CreatePurchaseOrder(ctx, application.PurchaseOrderRequest{SupplierID: supplier, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material, Quantity: "10"}}}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	po, err = service.SubmitPurchaseOrder(ctx, po.ID, "creator")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "checker", "po-reject-test-0001", "REJECT", po.Version, " "); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	first, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "checker", "po-reject-test-0001", "REJECT", po.Version, " Needs correction ")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := application.NewService(repo).ExecutePurchaseOrderAction(ctx, po.ID, "checker", "po-reject-test-0001", "REJECT", po.Version, "Needs correction")
	if err != nil || retry.RejectedAt == nil || !retry.RejectedAt.Equal(*first.RejectedAt) {
		t.Fatalf("reject retry changed evidence: %v", err)
	}
	if _, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "checker", "po-reject-test-0001", "REJECT", po.Version, "Different reason"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
}
