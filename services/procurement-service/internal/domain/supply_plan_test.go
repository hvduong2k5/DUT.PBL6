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

func TestSupplyPlanRevisionRequiresCurrentVersionAndRetainsDraftState(t *testing.T) {
	preview, err := domain.PreviewSupplyPlan(supplyPlanFixture(), time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewSupplyPlan("SP-20261107-REVISION", "supply-manager", preview)
	if err != nil {
		t.Fatal(err)
	}
	originalForecast := plan.Snapshot.Lines[0].ForecastQuantity
	preview.Lines[0].ForecastQuantity = "25"
	updated, revision, err := plan.Revise("supply-manager", 1, preview, time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.Version != 2 || updated.Status != domain.SupplyPlanDraft || revision.Revision != 2 || revision.Snapshot.Lines[0].ForecastQuantity != "25" {
		t.Fatalf("unexpected revised plan: updated=%+v revision=%+v", updated, revision)
	}
	if plan.Snapshot.Lines[0].ForecastQuantity != originalForecast {
		t.Fatalf("revising plan mutated the previous snapshot: %+v", plan.Snapshot.Lines[0])
	}
	if _, _, err := plan.Revise("supply-manager", 2, preview, time.Now()); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale version error=%v, want conflict", err)
	}
	plan.Status = domain.SupplyPlanStatus("APPROVED")
	if _, _, err := plan.Revise("supply-manager", 1, preview, time.Now()); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("non-draft revision error=%v, want invalid state", err)
	}
}

func TestSupplyPlanApprovalRequiresSeparationAndRecordsDecision(t *testing.T) {
	preview, err := domain.PreviewSupplyPlan(supplyPlanFixture(), time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewSupplyPlan("SP-20261107-APPROVAL", "creator", preview)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	if _, err := plan.Approve("approver", now); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("approve before submission error=%v, want invalid state", err)
	}
	plan, err = plan.SubmitForApproval("submitter", now)
	if err != nil || plan.Status != domain.SupplyPlanPendingApproval || plan.Version != 2 || plan.SubmittedBy != "submitter" {
		t.Fatalf("submit failed or fields missing: plan=%+v error=%v", plan, err)
	}
	if _, err := plan.Approve("submitter", now.Add(time.Minute)); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("submitter self-approval error=%v, want conflict", err)
	}
	plan, err = plan.Approve("manager", now.Add(time.Minute))
	if err != nil || plan.Status != domain.SupplyPlanApproved || plan.ApprovedBy != "manager" || plan.Version != 3 {
		t.Fatalf("approval decision missing: plan=%+v error=%v", plan, err)
	}
}

func TestSupplyPlanRejectRequiresReasonAndDifferentActor(t *testing.T) {
	preview, err := domain.PreviewSupplyPlan(supplyPlanFixture(), time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewSupplyPlan("SP-20261107-REJECT", "creator", preview)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	plan, err = plan.SubmitForApproval("submitter", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Reject("manager", " ", now.Add(time.Minute)); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("empty rejection reason error=%v, want invalid state", err)
	}
	plan, err = plan.Reject("manager", "forecast source is stale", now.Add(time.Minute))
	if err != nil || plan.Status != domain.SupplyPlanRejected || plan.RejectionReason != "forecast source is stale" {
		t.Fatalf("rejection decision missing: plan=%+v error=%v", plan, err)
	}
}

func TestPreviewSupplyPlanRecordsValidManualQuantityOverride(t *testing.T) {
	input := supplyPlanFixture()
	input.Lines[0].ManualOverrideQuantity = "12"
	input.Lines[0].ManualOverrideReason = "Limit purchase to available budget"
	preview, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	line := preview.Lines[0]
	if line.RecommendedOrderQuantity != "18.000000" || line.ProposedOrderQuantity != "12.000000" || !line.ManuallyOverridden ||
		line.ManualOverrideReason != "Limit purchase to available budget" || line.ProjectedInventory != "3.000000" ||
		line.RoundingExcess != "4.000000" || len(line.Warnings) != 2 || line.Warnings[0] != "MANUAL_QUANTITY_OVERRIDE" || line.Warnings[1] != "PROJECTED_BELOW_SAFETY_STOCK" {
		t.Fatalf("manual override was not captured in the calculation snapshot: %+v", line)
	}
}

func TestPreviewSupplyPlanRejectsInvalidManualOverrides(t *testing.T) {
	tests := []struct {
		name     string
		quantity string
		reason   string
	}{
		{name: "missing reason", quantity: "12"},
		{name: "below MOQ", quantity: "6", reason: "budget"},
		{name: "not pack multiple", quantity: "13", reason: "budget"},
		{name: "invalid quantity", quantity: "-1", reason: "budget"},
		{name: "reason without quantity", reason: "budget"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := supplyPlanFixture()
			input.Lines[0].ManualOverrideQuantity = test.quantity
			input.Lines[0].ManualOverrideReason = test.reason
			if _, err := domain.PreviewSupplyPlan(input, time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)); !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatalf("override error=%v, want invalid argument", err)
			}
		})
	}
}
