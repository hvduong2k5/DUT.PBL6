package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
)

func TestRevisionReplayNormalizesStringsWithoutMutatingInputOrRepricing(t *testing.T) {
	service, repo, supplier, material := makeApplicationFixture(t)
	ctx := context.Background()
	input := application.PurchaseOrderRequest{SupplierID: supplier, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material, Quantity: "10"}}}
	p, err := service.CreatePurchaseOrder(ctx, input, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if p, err = service.SubmitPurchaseOrder(ctx, p.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	if p, err = service.ApprovePurchaseOrder(ctx, p.ID, "checker"); err != nil {
		t.Fatal(err)
	}
	if p, err = service.MarkPurchaseOrderSent(ctx, p.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	if p, err = service.RecordSupplierResponse(ctx, p.ID, "buyer", domain.SupplierPOChangeRequested, "Reduce quantity"); err != nil {
		t.Fatal(err)
	}
	input.Notes = " Revision two "
	input.Lines[0].Quantity = " 8 "
	first, err := service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, input, " reviser ", "revision-unit-key-0001", p.Version, " Reduce quantity ")
	if err != nil {
		t.Fatal(err)
	}
	if input.Lines[0].Quantity != " 8 " || first.Notes != "Revision two" || first.Lines[0].Quantity != "8" {
		t.Fatal("normalization mutated caller input or stored whitespace")
	}
	repo.offers[supplier] = nil
	normalized := input
	normalized.Notes = "Revision two"
	normalized.Lines = []application.PurchaseOrderLineRequest{{MaterialID: material, Quantity: "8"}}
	for _, retryService := range []*application.Service{application.NewService(repo), application.NewService(&lateActionLookup{fakeRepository: repo})} {
		got, err := retryService.RevisePurchaseOrderWithIdempotency(ctx, p.ID, normalized, "reviser", "revision-unit-key-0001", p.Version, "Reduce quantity")
		if err != nil || got.Revision != 2 || got.Lines[0].ID != first.Lines[0].ID || got.TotalAmountMinor != first.TotalAmountMinor {
			t.Fatalf("replay tried to rebuild revision: %v", err)
		}
	}
	normalized.Lines[0].Quantity = "8.0"
	if _, err := service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, normalized, "reviser", "revision-unit-key-0001", p.Version, "Reduce quantity"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("quantity representation was not fingerprinted")
	}
	normalized.Lines[0].Quantity = "8"
	if _, err := service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, normalized, "reviser", "revision-fresh-stale-0001", p.Version, "Reduce quantity"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("stale revision command accepted")
	}
}
