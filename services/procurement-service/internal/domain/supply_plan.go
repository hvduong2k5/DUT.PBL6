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
	MaterialID       uuid.UUID
	MaterialCode     string
	MaterialName     string
	Unit             string
	ForecastQuantity string
	OnHandQuantity   string
	OpenPOQuantity   string
	SafetyStock      string
	MinimumOrderQty  string
	PackSize         string
	LeadTimeDays     int
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

type SupplyPlanLine struct {
	MaterialID            uuid.UUID `json:"material_id"`
	MaterialCode          string    `json:"material_code"`
	MaterialName          string    `json:"material_name"`
	Unit                  string    `json:"unit"`
	ForecastQuantity      string    `json:"forecast_quantity"`
	OnHandQuantity        string    `json:"on_hand_quantity"`
	OpenPOQuantity        string    `json:"open_po_quantity"`
	SafetyStock           string    `json:"safety_stock"`
	MinimumOrderQuantity  string    `json:"minimum_order_quantity"`
	PackSize              string    `json:"pack_size"`
	LeadTimeDays          int       `json:"lead_time_days"`
	RecommendedOrderDate  string    `json:"recommended_order_date"`
	NetRequirement        string    `json:"net_requirement"`
	ProposedOrderQuantity string    `json:"proposed_order_quantity"`
	RoundingExcess        string    `json:"rounding_excess"`
	ProjectedInventory    string    `json:"projected_inventory"`
	ProjectedAboveSafety  string    `json:"projected_above_safety_stock"`
	Warnings              []string  `json:"warnings"`
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
	projected := new(big.Rat).Sub(new(big.Rat).Add(new(big.Rat).Add(onHand, openPO), proposal), forecast)
	aboveSafety := new(big.Rat).Sub(projected, safetyStock)
	recommendedOrderDate := ""
	if proposal.Sign() > 0 {
		recommendedOrderDate = horizonStart.AddDate(0, 0, -input.LeadTimeDays).Format("2006-01-02")
	}
	return SupplyPlanLine{
		MaterialID: input.MaterialID, MaterialCode: input.MaterialCode, MaterialName: input.MaterialName, Unit: input.Unit,
		ForecastQuantity: forecast.FloatString(6), OnHandQuantity: onHand.FloatString(6), OpenPOQuantity: openPO.FloatString(6),
		SafetyStock: safetyStock.FloatString(6), MinimumOrderQuantity: minimum.FloatString(6), PackSize: pack.FloatString(6),
		LeadTimeDays: input.LeadTimeDays, RecommendedOrderDate: recommendedOrderDate,
		NetRequirement: net.FloatString(6), ProposedOrderQuantity: proposal.FloatString(6),
		RoundingExcess:     new(big.Rat).Sub(new(big.Rat).Set(proposal), net).FloatString(6),
		ProjectedInventory: projected.FloatString(6), ProjectedAboveSafety: aboveSafety.FloatString(6), Warnings: make([]string, 0),
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
