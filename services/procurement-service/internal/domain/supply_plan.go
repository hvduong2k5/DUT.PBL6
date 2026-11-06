package domain

import (
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SupplyPlanInput struct {
	HorizonStart      time.Time
	HorizonEnd        time.Time
	ForecastSource    string
	ForecastAsOf      time.Time
	InventorySource   string
	InventoryAsOf     time.Time
	StaleAfterMinutes int
	PolicyVersion     string
	Lines             []SupplyPlanLineInput
}

type SupplyPlanLineInput struct {
	MaterialID             uuid.UUID
	MaterialCode           string
	MaterialName           string
	Unit                   string
	ForecastQuantity       string
	OnHandQuantity         string
	OpenPOQuantity         string
	SafetyStock            string
	MinimumOrderQty        string
	PackSize               string
	LeadTimeDays           int
	ManualOverrideQuantity string
	ManualOverrideReason   string
}

type SupplyPlanPreview struct {
	HorizonStart      time.Time        `json:"horizon_start"`
	HorizonEnd        time.Time        `json:"horizon_end"`
	ForecastSource    string           `json:"forecast_source"`
	ForecastAsOf      time.Time        `json:"forecast_as_of"`
	InventorySource   string           `json:"inventory_source"`
	InventoryAsOf     time.Time        `json:"inventory_as_of"`
	StaleAfterMinutes int              `json:"stale_after_minutes"`
	PolicyVersion     string           `json:"policy_version"`
	Status            string           `json:"status"`
	Warnings          []string         `json:"warnings"`
	Lines             []SupplyPlanLine `json:"lines"`
}

type SupplyPlanStatus string

const (
	SupplyPlanDraft           SupplyPlanStatus = "DRAFT"
	SupplyPlanPendingApproval SupplyPlanStatus = "PENDING_APPROVAL"
	SupplyPlanApproved        SupplyPlanStatus = "APPROVED"
	SupplyPlanRejected        SupplyPlanStatus = "REJECTED"
	SupplyPlanConverted       SupplyPlanStatus = "CONVERTED"
)

type SupplyPlan struct {
	ID                       uuid.UUID         `json:"id"`
	Number                   string            `json:"number"`
	Revision                 int               `json:"revision"`
	Status                   SupplyPlanStatus  `json:"status"`
	Snapshot                 SupplyPlanPreview `json:"snapshot"`
	CreatedBy                string            `json:"created_by"`
	SubmittedBy              string            `json:"submitted_by,omitempty"`
	SubmittedAt              *time.Time        `json:"submitted_at,omitempty"`
	ApprovedBy               string            `json:"approved_by,omitempty"`
	ApprovedAt               *time.Time        `json:"approved_at,omitempty"`
	RejectedBy               string            `json:"rejected_by,omitempty"`
	RejectedAt               *time.Time        `json:"rejected_at,omitempty"`
	RejectionReason          string            `json:"rejection_reason,omitempty"`
	ConvertedPurchaseOrderID *uuid.UUID        `json:"converted_purchase_order_id,omitempty"`
	ConvertedAt              *time.Time        `json:"converted_at,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	Version                  int64             `json:"version"`
}

type SupplyPlanStatusEvent struct {
	SupplyPlanID uuid.UUID        `json:"supply_plan_id"`
	FromStatus   SupplyPlanStatus `json:"from_status"`
	ToStatus     SupplyPlanStatus `json:"to_status"`
	Action       string           `json:"action"`
	Actor        string           `json:"actor"`
	Reason       string           `json:"reason,omitempty"`
	Version      int64            `json:"version"`
	CreatedAt    time.Time        `json:"created_at"`
}

type SupplyPlanRevision struct {
	SupplyPlanID uuid.UUID         `json:"supply_plan_id"`
	Revision     int               `json:"revision"`
	Snapshot     SupplyPlanPreview `json:"snapshot"`
	CreatedBy    string            `json:"created_by"`
	CreatedAt    time.Time         `json:"created_at"`
}

func NewSupplyPlan(number, actor string, snapshot SupplyPlanPreview) (SupplyPlan, error) {
	number, actor = strings.TrimSpace(number), strings.TrimSpace(actor)
	if number == "" || actor == "" || len(snapshot.Lines) == 0 || snapshot.Status != "DRAFT_PREVIEW" && snapshot.Status != "INPUT_WARNING" {
		return SupplyPlan{}, ErrInvalidArgument
	}
	now := time.Now().UTC()
	return SupplyPlan{ID: uuid.New(), Number: number, Revision: 1, Status: SupplyPlanDraft, Snapshot: cloneSupplyPlanPreview(snapshot),
		CreatedBy: actor, CreatedAt: now, UpdatedAt: now, Version: 1}, nil
}

func (p SupplyPlan) Revise(actor string, expectedVersion int64, snapshot SupplyPlanPreview, now time.Time) (SupplyPlan, SupplyPlanRevision, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" || expectedVersion <= 0 || now.IsZero() || len(snapshot.Lines) == 0 ||
		(snapshot.Status != "DRAFT_PREVIEW" && snapshot.Status != "INPUT_WARNING") {
		return SupplyPlan{}, SupplyPlanRevision{}, ErrInvalidArgument
	}
	if p.Status != SupplyPlanDraft {
		return SupplyPlan{}, SupplyPlanRevision{}, ErrInvalidState
	}
	if p.Version != expectedVersion {
		return SupplyPlan{}, SupplyPlanRevision{}, ErrConflict
	}
	p.Revision++
	p.Version++
	p.Snapshot = cloneSupplyPlanPreview(snapshot)
	p.UpdatedAt = now.UTC()
	revision := SupplyPlanRevision{SupplyPlanID: p.ID, Revision: p.Revision, Snapshot: cloneSupplyPlanPreview(snapshot), CreatedBy: actor, CreatedAt: now.UTC()}
	return p, revision, nil
}

func (p SupplyPlan) SubmitForApproval(actor string, now time.Time) (SupplyPlan, error) {
	actor = strings.TrimSpace(actor)
	if p.Status != SupplyPlanDraft || actor == "" || now.IsZero() {
		return SupplyPlan{}, ErrInvalidState
	}
	p.Status = SupplyPlanPendingApproval
	p.SubmittedBy = actor
	p.SubmittedAt = timePointer(now)
	p.touch(now)
	return p, nil
}

func (p SupplyPlan) Approve(actor string, now time.Time) (SupplyPlan, error) {
	actor = strings.TrimSpace(actor)
	if p.Status != SupplyPlanPendingApproval || actor == "" || now.IsZero() {
		return SupplyPlan{}, ErrInvalidState
	}
	if actor == p.CreatedBy || actor == p.SubmittedBy {
		return SupplyPlan{}, ErrConflict
	}
	p.Status = SupplyPlanApproved
	p.ApprovedBy = actor
	p.ApprovedAt = timePointer(now)
	p.touch(now)
	return p, nil
}

func (p SupplyPlan) Reject(actor, reason string, now time.Time) (SupplyPlan, error) {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if p.Status != SupplyPlanPendingApproval || actor == "" || reason == "" || now.IsZero() {
		return SupplyPlan{}, ErrInvalidState
	}
	if actor == p.CreatedBy || actor == p.SubmittedBy {
		return SupplyPlan{}, ErrConflict
	}
	p.Status = SupplyPlanRejected
	p.RejectedBy = actor
	p.RejectedAt = timePointer(now)
	p.RejectionReason = reason
	p.touch(now)
	return p, nil
}

func (p SupplyPlan) ConvertToPurchaseOrder(actor string, purchaseOrderID uuid.UUID, expectedVersion int64, now time.Time) (SupplyPlan, error) {
	actor = strings.TrimSpace(actor)
	if p.Status != SupplyPlanApproved || actor == "" || purchaseOrderID == uuid.Nil || now.IsZero() {
		return SupplyPlan{}, ErrInvalidState
	}
	if p.Version != expectedVersion {
		return SupplyPlan{}, ErrConflict
	}
	p.Status = SupplyPlanConverted
	p.ConvertedPurchaseOrderID = &purchaseOrderID
	p.ConvertedAt = timePointer(now)
	p.touch(now)
	return p, nil
}

func (p *SupplyPlan) touch(now time.Time) {
	p.UpdatedAt = now.UTC()
	p.Version++
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}

func cloneSupplyPlanPreview(snapshot SupplyPlanPreview) SupplyPlanPreview {
	clone := snapshot
	clone.Warnings = append([]string(nil), snapshot.Warnings...)
	clone.Lines = append([]SupplyPlanLine(nil), snapshot.Lines...)
	for i := range clone.Lines {
		clone.Lines[i].Warnings = append([]string(nil), snapshot.Lines[i].Warnings...)
	}
	return clone
}

type SupplyPlanLine struct {
	MaterialID               uuid.UUID `json:"material_id"`
	MaterialCode             string    `json:"material_code"`
	MaterialName             string    `json:"material_name"`
	Unit                     string    `json:"unit"`
	ForecastQuantity         string    `json:"forecast_quantity"`
	OnHandQuantity           string    `json:"on_hand_quantity"`
	OpenPOQuantity           string    `json:"open_po_quantity"`
	SafetyStock              string    `json:"safety_stock"`
	MinimumOrderQuantity     string    `json:"minimum_order_quantity"`
	PackSize                 string    `json:"pack_size"`
	LeadTimeDays             int       `json:"lead_time_days"`
	RecommendedOrderDate     string    `json:"recommended_order_date"`
	NetRequirement           string    `json:"net_requirement"`
	RecommendedOrderQuantity string    `json:"recommended_order_quantity"`
	ProposedOrderQuantity    string    `json:"proposed_order_quantity"`
	ManuallyOverridden       bool      `json:"manually_overridden"`
	ManualOverrideReason     string    `json:"manual_override_reason,omitempty"`
	RoundingExcess           string    `json:"rounding_excess"`
	ProjectedInventory       string    `json:"projected_inventory"`
	ProjectedAboveSafety     string    `json:"projected_above_safety_stock"`
	Warnings                 []string  `json:"warnings"`
}

var nonNegativeSupplyQuantity = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,6})?$`)

func PreviewSupplyPlan(input SupplyPlanInput, now time.Time) (SupplyPlanPreview, error) {
	input.ForecastSource = strings.TrimSpace(input.ForecastSource)
	input.InventorySource = strings.TrimSpace(input.InventorySource)
	input.PolicyVersion = strings.TrimSpace(input.PolicyVersion)
	if input.HorizonStart.IsZero() || input.HorizonEnd.IsZero() || !input.HorizonStart.Before(input.HorizonEnd) ||
		input.ForecastSource == "" || input.InventorySource == "" || input.ForecastAsOf.IsZero() || input.InventoryAsOf.IsZero() ||
		input.StaleAfterMinutes <= 0 || input.PolicyVersion == "" || len(input.Lines) == 0 || now.IsZero() {
		return SupplyPlanPreview{}, ErrInvalidArgument
	}
	preview := SupplyPlanPreview{
		HorizonStart: input.HorizonStart.UTC(), HorizonEnd: input.HorizonEnd.UTC(),
		ForecastSource: input.ForecastSource, ForecastAsOf: input.ForecastAsOf.UTC(),
		InventorySource: input.InventorySource, InventoryAsOf: input.InventoryAsOf.UTC(),
		StaleAfterMinutes: input.StaleAfterMinutes, PolicyVersion: input.PolicyVersion,
		Status: "DRAFT_PREVIEW", Warnings: make([]string, 0), Lines: make([]SupplyPlanLine, 0, len(input.Lines)),
	}
	if now.Before(input.ForecastAsOf) || now.Sub(input.ForecastAsOf).Minutes() > float64(input.StaleAfterMinutes) {
		preview.Warnings = append(preview.Warnings, "FORECAST_STALE_OR_FUTURE")
	}
	if now.Before(input.InventoryAsOf) || now.Sub(input.InventoryAsOf).Minutes() > float64(input.StaleAfterMinutes) {
		preview.Warnings = append(preview.Warnings, "INVENTORY_STALE_OR_FUTURE")
	}
	seen := make(map[uuid.UUID]struct{}, len(input.Lines))
	for _, item := range input.Lines {
		line, err := calculateSupplyPlanLine(item, input.HorizonStart)
		if err != nil {
			return SupplyPlanPreview{}, err
		}
		if _, exists := seen[line.MaterialID]; exists {
			return SupplyPlanPreview{}, ErrInvalidArgument
		}
		seen[line.MaterialID] = struct{}{}
		preview.Lines = append(preview.Lines, line)
	}
	if len(preview.Warnings) > 0 {
		preview.Status = "INPUT_WARNING"
	}
	return preview, nil
}

func calculateSupplyPlanLine(input SupplyPlanLineInput, horizonStart time.Time) (SupplyPlanLine, error) {
	input.MaterialCode = strings.TrimSpace(input.MaterialCode)
	input.MaterialName = strings.TrimSpace(input.MaterialName)
	input.Unit = strings.ToUpper(strings.TrimSpace(input.Unit))
	forecast, forecastOK := parseSupplyQuantity(input.ForecastQuantity)
	onHand, onHandOK := parseSupplyQuantity(input.OnHandQuantity)
	openPO, openPOOK := parseSupplyQuantity(input.OpenPOQuantity)
	safetyStock, safetyOK := parseSupplyQuantity(input.SafetyStock)
	minimum, minimumOK := parseSupplyQuantity(input.MinimumOrderQty)
	packSize := strings.TrimSpace(input.PackSize)
	if packSize == "" {
		packSize = "1"
	}
	pack, packOK := parseSupplyQuantity(packSize)
	if input.MaterialID == uuid.Nil || input.MaterialCode == "" || input.MaterialName == "" || input.Unit == "" ||
		!forecastOK || !onHandOK || !openPOOK || !safetyOK || !minimumOK || !packOK || minimum.Sign() <= 0 || pack.Sign() <= 0 || input.LeadTimeDays < 0 {
		return SupplyPlanLine{}, ErrInvalidArgument
	}
	net := new(big.Rat).Sub(new(big.Rat).Add(forecast, safetyStock), new(big.Rat).Add(onHand, openPO))
	if net.Sign() < 0 {
		net.SetInt64(0)
	}
	proposal := new(big.Rat)
	if net.Sign() > 0 {
		proposal.Set(net)
		if proposal.Cmp(minimum) < 0 {
			proposal.Set(minimum)
		}
		proposal = ceilToMultiple(proposal, pack)
	}
	recommendedProposal := new(big.Rat).Set(proposal)
	manualOverrideQuantity := strings.TrimSpace(input.ManualOverrideQuantity)
	manualOverrideReason := strings.TrimSpace(input.ManualOverrideReason)
	manuallyOverridden := manualOverrideQuantity != "" || manualOverrideReason != ""
	if manuallyOverridden {
		if manualOverrideQuantity == "" || manualOverrideReason == "" || len(manualOverrideReason) > 500 {
			return SupplyPlanLine{}, ErrInvalidArgument
		}
		override, valid := parseSupplyQuantity(manualOverrideQuantity)
		if !valid {
			return SupplyPlanLine{}, ErrInvalidArgument
		}
		if override.Sign() > 0 && (override.Cmp(minimum) < 0 || !new(big.Rat).Quo(override, pack).IsInt()) {
			return SupplyPlanLine{}, ErrInvalidArgument
		}
		proposal.Set(override)
	}
	projected := new(big.Rat).Sub(new(big.Rat).Add(new(big.Rat).Add(onHand, openPO), proposal), forecast)
	aboveSafety := new(big.Rat).Sub(projected, safetyStock)
	recommendedOrderDate := ""
	if proposal.Sign() > 0 {
		recommendedOrderDate = horizonStart.AddDate(0, 0, -input.LeadTimeDays).Format("2006-01-02")
	}
	lineWarnings := make([]string, 0, 1)
	if manuallyOverridden {
		lineWarnings = append(lineWarnings, "MANUAL_QUANTITY_OVERRIDE")
	}
	if aboveSafety.Sign() < 0 {
		lineWarnings = append(lineWarnings, "PROJECTED_BELOW_SAFETY_STOCK")
	}
	return SupplyPlanLine{
		MaterialID: input.MaterialID, MaterialCode: input.MaterialCode, MaterialName: input.MaterialName, Unit: input.Unit,
		ForecastQuantity: forecast.FloatString(6), OnHandQuantity: onHand.FloatString(6), OpenPOQuantity: openPO.FloatString(6),
		SafetyStock: safetyStock.FloatString(6), MinimumOrderQuantity: minimum.FloatString(6), PackSize: pack.FloatString(6),
		LeadTimeDays: input.LeadTimeDays, RecommendedOrderDate: recommendedOrderDate,
		NetRequirement: net.FloatString(6), RecommendedOrderQuantity: recommendedProposal.FloatString(6), ProposedOrderQuantity: proposal.FloatString(6),
		ManuallyOverridden: manuallyOverridden, ManualOverrideReason: manualOverrideReason,
		RoundingExcess:     new(big.Rat).Sub(new(big.Rat).Set(recommendedProposal), net).FloatString(6),
		ProjectedInventory: projected.FloatString(6), ProjectedAboveSafety: aboveSafety.FloatString(6), Warnings: lineWarnings,
	}, nil
}

func parseSupplyQuantity(value string) (*big.Rat, bool) {
	value = strings.TrimSpace(value)
	if !nonNegativeSupplyQuantity.MatchString(value) {
		return nil, false
	}
	parsed, ok := new(big.Rat).SetString(value)
	return parsed, ok && parsed.Sign() >= 0
}

func ceilToMultiple(value, multiple *big.Rat) *big.Rat {
	quotient := new(big.Rat).Quo(value, multiple)
	whole, remainder := new(big.Int).QuoRem(quotient.Num(), quotient.Denom(), new(big.Int))
	if remainder.Sign() > 0 {
		whole.Add(whole, big.NewInt(1))
	}
	return new(big.Rat).Mul(new(big.Rat).SetInt(whole), multiple)
}
