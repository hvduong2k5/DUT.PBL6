package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/repository/postgres"
)

func assertPODraftEditE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool, repo *postgres.Repository, service *application.Service, draft domain.PurchaseOrder, supplierID, materialID, offerID uuid.UUID) {
	t.Helper()
	input := application.PurchaseOrderRequest{SupplierID: supplierID, Notes: "Draft correction", Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "7"}}}
	const actor = "draft-editor"
	const key = "draft-edit-pg-0001"
	if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_draft_command() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='EDIT_DRAFT' THEN RAISE EXCEPTION 'injected draft edit journal failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER reject_test_draft_command BEFORE INSERT ON purchase_order_action_results FOR EACH ROW EXECUTE FUNCTION reject_test_draft_command()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdatePurchaseOrderDraftWithIdempotency(ctx, draft.ID, input, actor, key, draft.Version); err == nil || !strings.Contains(err.Error(), "injected draft edit journal failure") {
		t.Fatalf("draft journal fault: %v", err)
	}
	current, err := repo.GetPurchaseOrder(ctx, draft.ID)
	if err != nil || current.Version != draft.Version || current.Lines[0].ID != draft.Lines[0].ID || current.Revision != 3 {
		t.Fatal("failed edit leaked draft changes")
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='DRAFT_UPDATED'`, draft.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed edit leaked history")
	}
	if _, err := db.Exec(ctx, `DROP TRIGGER reject_test_draft_command ON purchase_order_action_results; DROP FUNCTION reject_test_draft_command()`); err != nil {
		t.Fatal(err)
	}
	type result struct {
		po  domain.PurchaseOrder
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			p, err := application.NewService(postgres.New(db)).UpdatePurchaseOrderDraftWithIdempotency(ctx, draft.ID, input, actor, key, draft.Version)
			results <- result{p, err}
		}()
	}
	close(start)
	var edited domain.PurchaseOrder
	for i := 0; i < 2; i++ {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if i == 1 && !reflect.DeepEqual(edited, got.po) {
			t.Fatal("duplicate edit returned different results")
		}
		edited = got.po
	}
	if edited.Version != draft.Version+1 || edited.Revision != 3 || edited.CreatedBy != draft.CreatedBy || edited.ID != draft.ID || edited.Number != draft.Number {
		t.Fatal("draft edit lost revision or identity")
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='DRAFT_UPDATED'`, draft.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("edit duplicate history")
	}
	// Competing new commands at one version must not silently overwrite each other.
	results = make(chan result, 2)
	start = make(chan struct{})
	for _, quantity := range []string{"8", "9"} {
		go func(q string) {
			<-start
			request := application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: q}}}
			p, err := application.NewService(postgres.New(db)).UpdatePurchaseOrderDraftWithIdempotency(ctx, draft.ID, request, actor, "draft-edit-competing-"+q+"-0001", edited.Version)
			results <- result{p, err}
		}(quantity)
	}
	close(start)
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		got := <-results
		if got.err == nil {
			wins++
		} else if errors.Is(got.err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(got.err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal("competing edits lost optimistic concurrency")
	}
	archive, err := repo.ListPurchaseOrderRevisions(ctx, draft.ID, 100, 0)
	if err != nil || len(archive) != 2 {
		t.Fatal("draft edit fabricated an archive revision")
	}
	if _, err := service.SubmitPurchaseOrder(ctx, draft.ID, draft.CreatedBy); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetOfferActive(ctx, supplierID, offerID, false); err != nil {
		t.Fatal(err)
	}
	replay, err := application.NewService(postgres.New(db)).UpdatePurchaseOrderDraftWithIdempotency(ctx, draft.ID, input, actor, key, draft.Version)
	if err != nil || !reflect.DeepEqual(replay, edited) {
		t.Fatalf("edit replay repriced or advanced PO: %v", err)
	}
	if _, err := service.SetOfferActive(ctx, supplierID, offerID, true); err != nil {
		t.Fatal(err)
	}
	// Real list filtering/pagination, including equal-time UUID ordering.
	var tieIDs []uuid.UUID
	for i := 0; i < 2; i++ {
		p, err := service.CreatePurchaseOrder(ctx, input, "list-buyer")
		if err != nil {
			t.Fatal(err)
		}
		tieIDs = append(tieIDs, p.ID)
	}
	if _, err := db.Exec(ctx, `UPDATE purchase_orders SET created_at='2026-01-01T00:00:00Z' WHERE id=ANY($1::uuid[])`, tieIDs); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListPurchaseOrders(ctx, supplierID, domain.PurchaseOrderDraft, 100, 0)
	if err != nil || len(list) < 2 {
		t.Fatal("filtered list missing drafts")
	}
	for i, p := range list {
		if p.SupplierID != supplierID || p.Status != domain.PurchaseOrderDraft {
			t.Fatal("list filter mismatch")
		}
		if i > 0 && (list[i-1].CreatedAt.Before(p.CreatedAt) || (list[i-1].CreatedAt.Equal(p.CreatedAt) && list[i-1].ID.String() > p.ID.String())) {
			t.Fatal("unstable list ordering")
		}
	}
	page, err := repo.ListPurchaseOrders(ctx, supplierID, domain.PurchaseOrderDraft, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != list[1].ID {
		t.Fatal("list pagination mismatch")
	}
	empty, err := repo.ListPurchaseOrders(ctx, uuid.New(), "", 100, 0)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("unknown supplier should yield empty page")
	}
}
