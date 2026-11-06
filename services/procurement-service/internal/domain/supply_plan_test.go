package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func supplyPlanFixture() domain.SupplyPlanInput {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	return domain.SupplyPlanInput{
		HorizonStart: now.AddDate(0, 0, 1), HorizonEnd: now.AddDate(0, 1, 0),
		ForecastSource: "forecast-snapshot-42", ForecastAsOf: now.Add(-time.Hour),
		InventorySource: "inventory-snapshot-8", InventoryAsOf: now.Add(-30 * time.Minute),
		StaleAfterMinutes: 180, PolicyVersion: "procurement-policy-v1",
		Lines: []domain.SupplyPlanLineInput{{MaterialID: uuid.New(), MaterialCode: "RICE", MaterialName: "Rice", Unit: "kg",
			ForecastQuantity: "20", OnHandQuantity: "8", OpenPOQuantity: "3", SafetyStock: "5", MinimumOrderQty: "10", PackSize: "6", LeadTimeDays: 4}},
	}
}

func TestPreviewSupplyPlanCalculatesNetNeedMOQAndPackRounding(t *testing.T) {
	input := supplyPlanFixture()
	preview, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Status != "DRAFT_PREVIEW" || len(preview.Lines) != 1 {
		t.Fatalf("unexpected preview status or line count: %+v", preview)
	}
	line := preview.Lines[0]
	if line.NetRequirement != "14.000000" || line.ProposedOrderQuantity != "18.000000" || line.RoundingExcess != "4.000000" || line.ProjectedInventory != "9.000000" || line.ProjectedAboveSafety != "4.000000" || line.RecommendedOrderDate != "2026-11-04" {
		t.Fatalf("incorrect plan arithmetic: %+v", line)
	}
}

func TestPreviewSupplyPlanReportsStaleAndFutureInputs(t *testing.T) {
	input := supplyPlanFixture()
	input.ForecastAsOf = time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	input.InventoryAsOf = time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	preview, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Status != "INPUT_WARNING" || len(preview.Warnings) != 2 {
		t.Fatalf("stale/future inputs were not surfaced: %+v", preview)
	}
}

func TestPreviewSupplyPlanDoesNotRecommendPurchaseWhenStockAndOpenPOCoverNeed(t *testing.T) {
	input := supplyPlanFixture()
	input.Lines[0].ForecastQuantity = "20"
	input.Lines[0].OnHandQuantity = "50"
	input.Lines[0].OpenPOQuantity = "0"
	preview, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Lines[0].NetRequirement != "0.000000" || preview.Lines[0].ProposedOrderQuantity != "0.000000" {
		t.Fatalf("unexpected purchase recommendation for covered demand: %+v", preview.Lines[0])
	}
}

func TestPreviewSupplyPlanRejectsDuplicateOrInvalidLines(t *testing.T) {
	input := supplyPlanFixture()
	input.Lines = append(input.Lines, input.Lines[0])
	if _, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("duplicate line error = %v, want invalid argument", err)
	}
	input = supplyPlanFixture()
	input.Lines[0].PackSize = "0"
	if _, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("zero pack size error = %v, want invalid argument", err)
	}
}

func TestNewSupplyPlanCreatesDraftRevisionFromSnapshot(t *testing.T) {
	preview, err := domain.PreviewSupplyPlan(supplyPlanFixture(), time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewSupplyPlan("SP-20261107-1234ABCD", "supply-manager", preview)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != domain.SupplyPlanDraft || plan.Revision != 1 || plan.Version != 1 || plan.Snapshot.Lines[0].ProposedOrderQuantity != "18.000000" {
		t.Fatalf("unexpected new supply plan: %+v", plan)
	}
}
