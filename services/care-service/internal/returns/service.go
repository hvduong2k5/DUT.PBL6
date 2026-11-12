package returns

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
	"github.com/omamx/care-service/internal/shipping"
)

type Service struct {
	repository Repository
	orders     OrderContextProvider
	shipping   *shipping.Client
	now        func() time.Time
}

func NewService(repository Repository, orders OrderContextProvider, shippingClient *shipping.Client) *Service {
	return &Service{repository: repository, orders: orders, shipping: shippingClient, now: time.Now}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Case, bool, error) {
	input.OrderID = strings.TrimSpace(input.OrderID)
	input.Resolution = strings.TrimSpace(input.Resolution)
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	input.CustomerNote = strings.TrimSpace(input.CustomerNote)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ActorID == uuid.Nil || input.OrderID == "" || len(input.OrderID) > 100 {
		return Case{}, false, fmt.Errorf("%w: customer and order are required", domain.ErrInvalidArgument)
	}
	if input.Resolution != "RETURN_AND_REFUND" && input.Resolution != "REFUND_WITHOUT_RETURN" {
		return Case{}, false, fmt.Errorf("%w: unsupported return resolution", domain.ErrInvalidArgument)
	}
	if input.Resolution == "REFUND_WITHOUT_RETURN" {
		return Case{}, false, fmt.Errorf("%w: refund without return is a staff-only exception", domain.ErrForbidden)
	}
	if input.ReasonCode == "" || len(input.ReasonCode) > 50 || len(input.CustomerNote) > 2000 {
		return Case{}, false, fmt.Errorf("%w: reason code or note is invalid", domain.ErrInvalidArgument)
	}
	if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
		return Case{}, false, err
	}
	if len(input.Lines) == 0 || len(input.Lines) > 50 || len(input.EvidenceFileIDs) > 6 {
		return Case{}, false, fmt.Errorf("%w: return must contain 1-50 lines and at most 6 evidence files", domain.ErrInvalidArgument)
	}
	for _, fileID := range input.EvidenceFileIDs {
		if fileID == uuid.Nil {
			return Case{}, false, fmt.Errorf("%w: evidence file ID must be a UUID", domain.ErrInvalidArgument)
		}
	}
	requestHash := createRequestDigest(input)
	if replay, found, err := s.repository.ReplayCreate(ctx, input.ActorID, input.IdempotencyKey, requestHash); err != nil {
		return Case{}, false, err
	} else if found {
		return replay, false, nil
	}
	if s.orders == nil {
		return Case{}, false, domain.ErrDependencyUnavailable
	}
	order, err := s.orders.GetReturnContext(ctx, input.OrderID)
	if err != nil {
		return Case{}, false, err
	}
	now := s.now().UTC()
	if order.OrderID != input.OrderID || order.CustomerID == uuid.Nil {
		return Case{}, false, fmt.Errorf("%w: Order returned an invalid identity snapshot", domain.ErrDependencyUnavailable)
	}
	if order.CustomerID != input.ActorID {
		return Case{}, false, domain.ErrNotFound
	}
	if order.ContractVersion == "" || len(order.ContractVersion) > 50 || order.CapturedAt.IsZero() || order.CapturedAt.After(now) || order.ReturnEligibleAt.IsZero() || order.EligibilityExpiresAt.Before(order.ReturnEligibleAt) {
		return Case{}, false, fmt.Errorf("%w: Order returned an incomplete eligibility snapshot", domain.ErrDependencyUnavailable)
	}
	if now.Before(order.ReturnEligibleAt) || now.After(order.EligibilityExpiresAt) {
		return Case{}, false, fmt.Errorf("%w: the Order return eligibility window is not open", domain.ErrConflict)
	}
	if err := validateShippingContext(order); err != nil {
		return Case{}, false, fmt.Errorf("%w: Order return context cannot create a pickup", domain.ErrDependencyUnavailable)
	}
	if len(order.Lines) == 0 || len(order.Lines) > 200 {
		return Case{}, false, fmt.Errorf("%w: Order returned an invalid customer or line collection", domain.ErrDependencyUnavailable)
	}
	orderSnapshot, err := json.Marshal(order)
	if err != nil || len(orderSnapshot) > 65536 {
		return Case{}, false, fmt.Errorf("%w: Order return snapshot exceeds the supported size", domain.ErrDependencyUnavailable)
	}
	lineMap := make(map[string]OrderLine, len(order.Lines))
	for _, line := range order.Lines {
		if line.OrderLineID == "" || len(line.OrderLineID) > 100 || len(line.ProductID) > 100 || len(line.SKU) > 100 || len(line.ProductName) > 255 || line.PurchasedQuantity < 1 || line.PreviouslyCommittedReturnQuantity < 0 || line.PreviouslyCommittedReturnQuantity > line.PurchasedQuantity || line.RefundableUnitCeilingMinor < 0 || !validCurrency(line.Currency) {
			return Case{}, false, fmt.Errorf("%w: Order returned an invalid line snapshot", domain.ErrDependencyUnavailable)
		}
		lineSnapshot, marshalErr := json.Marshal(line)
		if marshalErr != nil || len(lineSnapshot) > 32768 {
			return Case{}, false, fmt.Errorf("%w: Order returned an oversized line snapshot", domain.ErrDependencyUnavailable)
		}
		if _, duplicate := lineMap[line.OrderLineID]; duplicate {
			return Case{}, false, fmt.Errorf("%w: Order returned duplicate line identities", domain.ErrDependencyUnavailable)
		}
		lineMap[line.OrderLineID] = line
	}
	requested := make([]CaseLine, 0, len(input.Lines))
	seen := make(map[string]struct{}, len(input.Lines))
	for i := range input.Lines {
		input.Lines[i].OrderLineID = strings.TrimSpace(input.Lines[i].OrderLineID)
	}
	for _, item := range input.Lines {
		line, exists := lineMap[item.OrderLineID]
		if !exists || item.Quantity < 1 || item.Quantity > line.PurchasedQuantity-line.PreviouslyCommittedReturnQuantity {
			return Case{}, false, fmt.Errorf("%w: return quantity exceeds the Order-authoritative remaining quantity", domain.ErrConflict)
		}
		if _, duplicate := seen[item.OrderLineID]; duplicate {
			return Case{}, false, fmt.Errorf("%w: duplicate order line", domain.ErrInvalidArgument)
		}
		seen[item.OrderLineID] = struct{}{}
		if line.RefundableUnitCeilingMinor > math.MaxInt64/int64(item.Quantity) {
			return Case{}, false, fmt.Errorf("%w: Order refund ceiling overflows for requested quantity", domain.ErrDependencyUnavailable)
		}
		requested = append(requested, CaseLine{
			ID: idgen.New(), OrderLineID: line.OrderLineID, ProductID: line.ProductID,
			SKU: line.SKU, ProductName: line.ProductName, PurchasedQuantity: line.PurchasedQuantity,
			RequestedQuantity: item.Quantity, RefundableCeiling: line.RefundableUnitCeilingMinor * int64(item.Quantity),
			RefundableUnitCeiling: line.RefundableUnitCeilingMinor, Currency: line.Currency,
		})
	}
	caseID := idgen.New()
	caseValue := Case{
		ID: caseID, Code: returnCaseCode(caseID), CustomerID: input.ActorID, OrderID: order.OrderID,
		OrderNumber: order.OrderNumber, Status: domain.ReturnCaseRequested, Resolution: input.Resolution,
		ReasonCode: input.ReasonCode, CustomerNote: input.CustomerNote, Version: 1, Lines: requested,
		EvidenceFileIDs: append([]uuid.UUID(nil), input.EvidenceFileIDs...), OrderContext: order,
		CreatedAt: now, UpdatedAt: now,
	}
	return s.repository.Create(ctx, CreateRecord{
		Case: caseValue, OrderContext: order, IdempotencyKey: input.IdempotencyKey,
		RequestHash: requestHash, IdempotencyExpiry: now.Add(24 * time.Hour), RequestID: input.RequestID,
	})
}

func (s *Service) GetOwned(ctx context.Context, customerID, caseID uuid.UUID) (Case, error) {
	if customerID == uuid.Nil || caseID == uuid.Nil {
		return Case{}, domain.ErrNotFound
	}
	return s.repository.GetOwned(ctx, customerID, caseID)
}

func (s *Service) ListOwned(ctx context.Context, customerID uuid.UUID, status domain.ReturnCaseStatus, page, pageSize int) ([]Case, int, error) {
	if customerID == uuid.Nil {
		return nil, 0, domain.ErrForbidden
	}
	if status != "" && !status.Valid() {
		return nil, 0, fmt.Errorf("%w: unsupported return case status", domain.ErrInvalidArgument)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repository.ListOwned(ctx, customerID, status, page, pageSize)
}

func (s *Service) Get(ctx context.Context, caseID uuid.UUID) (Case, error) {
	if caseID == uuid.Nil {
		return Case{}, domain.ErrNotFound
	}
	return s.repository.Get(ctx, caseID)
}

func (s *Service) List(ctx context.Context, status domain.ReturnCaseStatus, page, pageSize int) ([]Case, int, error) {
	if status != "" && !status.Valid() {
		return nil, 0, fmt.Errorf("%w: unsupported return case status", domain.ErrInvalidArgument)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repository.List(ctx, status, page, pageSize)
}

func (s *Service) StatusCounts(ctx context.Context) (map[domain.ReturnCaseStatus]int, error) {
	return s.repository.StatusCounts(ctx)
}

func (s *Service) Action(ctx context.Context, input ActionInput) (Case, bool, error) {
	input.Action = strings.ToUpper(strings.TrimSpace(input.Action))
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	input.Note = strings.TrimSpace(input.Note)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ActorID == uuid.Nil || input.CaseID == uuid.Nil || input.ExpectedVersion < 1 || input.Note == "" || len(input.Note) > 2000 {
		return Case{}, false, fmt.Errorf("%w: actor, case, expected revision, and reason are required", domain.ErrInvalidArgument)
	}
	if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
		return Case{}, false, err
	}
	now := s.now().UTC()
	var decision *Decision
	if input.Action == "DECIDE" {
		if input.ReasonCode == "" || len(input.ReasonCode) > 50 {
			return Case{}, false, fmt.Errorf("%w: decision reason code is required", domain.ErrInvalidArgument)
		}
		current, err := s.repository.Get(ctx, input.CaseID)
		if err != nil {
			return Case{}, false, err
		}
		if current.Version == input.ExpectedVersion && current.Status == domain.ReturnCaseReviewing {
			decision, err = buildDecision(input, current, now)
			if err != nil {
				return Case{}, false, err
			}
		}
	} else if input.Action != "START_REVIEW" && input.Action != "REQUEST_INFORMATION" {
		return Case{}, false, fmt.Errorf("%w: unsupported return case action", domain.ErrInvalidArgument)
	}
	updated, applied, err := s.repository.Action(ctx, ActionRecord{
		CaseID: input.CaseID, ActorID: input.ActorID, ExpectedVersion: input.ExpectedVersion,
		Action: input.Action, ReasonCode: input.ReasonCode, Note: input.Note, Decision: decision,
		IdempotencyKey: input.IdempotencyKey, RequestHash: actionRequestDigest(input),
		IdempotencyExpiry: now.Add(24 * time.Hour), RequestID: input.RequestID, At: now,
	})
	if err != nil {
		return Case{}, false, err
	}
	if updated.Decision != nil && updated.Decision.ReturnRequired && updated.ReverseShipmentID == "" {
		if s.shipping == nil {
			return updated, applied, domain.ErrDependencyUnavailable
		}
		shipment, callErr := s.shipping.CreateForApprovedDecision(ctx, toShippingDecision(updated))
		if callErr != nil {
			return updated, applied, fmt.Errorf("%w: create reverse shipment: %v", domain.ErrDependencyUnavailable, callErr)
		}
		updated, err = s.repository.AttachReverseShipment(ctx, updated.ID, updated.Decision.ID, shipment.ID, s.now().UTC())
		if err != nil {
			return Case{}, false, err
		}
	}
	return updated, applied, nil
}

func buildDecision(input ActionInput, current Case, now time.Time) (*Decision, error) {
	if len(input.Lines) != len(current.Lines) || len(input.Lines) == 0 {
		return nil, fmt.Errorf("%w: decision must include every case line", domain.ErrInvalidArgument)
	}
	caseLines := make(map[uuid.UUID]CaseLine, len(current.Lines))
	var totalCeiling int64
	currency := ""
	for _, line := range current.Lines {
		caseLines[line.ID] = line
		if currency == "" {
			currency = line.Currency
		} else if currency != line.Currency {
			return nil, fmt.Errorf("%w: cross-currency return decisions are not supported", domain.ErrConflict)
		}
		if line.RequestedQuantity < 1 || line.RefundableCeiling > math.MaxInt64-totalCeiling || line.RefundableUnitCeiling > math.MaxInt64/int64(line.RequestedQuantity) || line.RefundableUnitCeiling*int64(line.RequestedQuantity) != line.RefundableCeiling {
			return nil, fmt.Errorf("%w: total refund ceiling overflows", domain.ErrConflict)
		}
		totalCeiling += line.RefundableCeiling
	}
	seen := make(map[uuid.UUID]struct{}, len(input.Lines))
	lines := make([]DecisionLineInput, 0, len(input.Lines))
	approvedLines := 0
	var total int64
	for _, item := range input.Lines {
		line, ok := caseLines[item.ReturnCaseLineID]
		if !ok || item.ApprovedQuantity < 0 || item.ApprovedQuantity > line.RequestedQuantity || item.ApprovedAmountMinor < 0 || line.RefundableUnitCeiling > 0 && int64(item.ApprovedQuantity) > math.MaxInt64/line.RefundableUnitCeiling {
			return nil, fmt.Errorf("%w: approved line exceeds Order-authoritative quantity or refund ceiling", domain.ErrConflict)
		}
		approvedLineCeiling := line.RefundableUnitCeiling * int64(item.ApprovedQuantity)
		if item.ApprovedAmountMinor > approvedLineCeiling {
			return nil, fmt.Errorf("%w: approved line exceeds Order-authoritative quantity or refund ceiling", domain.ErrConflict)
		}
		if _, duplicate := seen[item.ReturnCaseLineID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate decision line", domain.ErrInvalidArgument)
		}
		seen[item.ReturnCaseLineID] = struct{}{}
		item.ReasonCode = strings.TrimSpace(item.ReasonCode)
		if item.ApprovedQuantity == 0 && item.ApprovedAmountMinor != 0 {
			return nil, fmt.Errorf("%w: rejected line cannot have a refund amount", domain.ErrInvalidArgument)
		}
		if item.ApprovedQuantity > 0 {
			approvedLines++
			if item.ReasonCode == "" {
				item.ReasonCode = input.ReasonCode
			}
		} else if item.ReasonCode == "" {
			item.ReasonCode = "NOT_APPROVED"
		}
		if item.ApprovedAmountMinor > math.MaxInt64-total {
			return nil, fmt.Errorf("%w: approved refund total overflows", domain.ErrConflict)
		}
		total += item.ApprovedAmountMinor
		lines = append(lines, item)
	}
	if len(seen) != len(caseLines) {
		return nil, fmt.Errorf("%w: decision must include each case line exactly once", domain.ErrInvalidArgument)
	}
	outcome := domain.ReturnCaseRejected
	if approvedLines > 0 {
		outcome = domain.ReturnCasePartiallyApproved
		allQuantitiesApproved := true
		for _, line := range lines {
			caseLine := caseLines[line.ReturnCaseLineID]
			if line.ApprovedQuantity != caseLine.RequestedQuantity {
				allQuantitiesApproved = false
				break
			}
		}
		if allQuantitiesApproved {
			outcome = domain.ReturnCaseApproved
		}
		if total < 1 {
			return nil, fmt.Errorf("%w: an approved return must have a positive refund amount", domain.ErrInvalidArgument)
		}
	}
	if outcome == domain.ReturnCaseRejected && input.ReturnRequired {
		return nil, fmt.Errorf("%w: rejected case cannot require returned goods", domain.ErrInvalidArgument)
	}
	return &Decision{
		ID: idgen.New(), Version: current.Version + 1, Outcome: outcome, ActorID: input.ActorID,
		ReasonCode: input.ReasonCode, Reason: input.Note, ReturnRequired: input.ReturnRequired && approvedLines > 0,
		ApprovedTotalMinor: total, AuthoritativeCeiling: totalCeiling, Currency: currency, Lines: lines, DecidedAt: now,
	}, nil
}

func validateShippingContext(order OrderContext) error {
	address := order.PickupAddress
	destination := order.Destination
	if address.Name == "" || address.Phone == "" || address.Street == "" || address.Ward == "" || address.Province == "" ||
		destination.Name == "" || destination.Phone == "" || destination.Street == "" || destination.Ward == "" || destination.Province == "" ||
		order.Parcel.WeightGrams < 1 || order.Parcel.LengthCM < 1 || order.Parcel.WidthCM < 1 || order.Parcel.HeightCM < 1 {
		return fmt.Errorf("missing pickup/destination/parcel data")
	}
	return nil
}

func toShippingDecision(value Case) shipping.ApprovedDecision {
	decision := value.Decision
	return shipping.ApprovedDecision{
		CaseID: value.ID.String(), DecisionVersion: decision.Version, Status: decision.Outcome,
		ReturnRequired: decision.ReturnRequired, OrderID: value.OrderID,
		PackageID: "return-package:" + value.ID.String(), ReasonCode: decision.ReasonCode,
		PickupAddress: shipping.Address{
			Name: value.OrderContext.PickupAddress.Name, Phone: value.OrderContext.PickupAddress.Phone,
			Street: value.OrderContext.PickupAddress.Street, Ward: value.OrderContext.PickupAddress.Ward,
			District: value.OrderContext.PickupAddress.District, Province: value.OrderContext.PickupAddress.Province,
			CountryCode: value.OrderContext.PickupAddress.CountryCode, PostalCode: value.OrderContext.PickupAddress.PostalCode,
		},
		Destination: shipping.Address{
			Name: value.OrderContext.Destination.Name, Phone: value.OrderContext.Destination.Phone,
			Street: value.OrderContext.Destination.Street, Ward: value.OrderContext.Destination.Ward,
			District: value.OrderContext.Destination.District, Province: value.OrderContext.Destination.Province,
			CountryCode: value.OrderContext.Destination.CountryCode, PostalCode: value.OrderContext.Destination.PostalCode,
		},
		Parcel: shipping.Parcel{
			WeightGrams: value.OrderContext.Parcel.WeightGrams, LengthCM: value.OrderContext.Parcel.LengthCM,
			WidthCM: value.OrderContext.Parcel.WidthCM, HeightCM: value.OrderContext.Parcel.HeightCM,
			Fragile: value.OrderContext.Parcel.Fragile,
		},
	}
}

func digest(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func createRequestDigest(input CreateInput) string {
	return digest(struct {
		ActorID         uuid.UUID       `json:"actor_id"`
		OrderID         string          `json:"order_id"`
		Resolution      string          `json:"resolution"`
		ReasonCode      string          `json:"reason_code"`
		CustomerNote    string          `json:"customer_note"`
		Lines           []RequestedLine `json:"lines"`
		EvidenceFileIDs []uuid.UUID     `json:"evidence_file_ids"`
	}{input.ActorID, input.OrderID, input.Resolution, input.ReasonCode, input.CustomerNote, input.Lines, input.EvidenceFileIDs})
}

func actionRequestDigest(input ActionInput) string {
	return digest(struct {
		ActorID         uuid.UUID           `json:"actor_id"`
		CaseID          uuid.UUID           `json:"case_id"`
		Action          string              `json:"action"`
		ExpectedVersion int                 `json:"expected_version"`
		ReasonCode      string              `json:"reason_code"`
		Note            string              `json:"note"`
		ReturnRequired  bool                `json:"return_required"`
		Lines           []DecisionLineInput `json:"lines"`
	}{input.ActorID, input.CaseID, input.Action, input.ExpectedVersion, input.ReasonCode, input.Note, input.ReturnRequired, input.Lines})
}

func validateIdempotencyKey(value string) error {
	if value == "" || len(value) > 200 {
		return fmt.Errorf("%w: Idempotency-Key is required and must not exceed 200 characters", domain.ErrInvalidArgument)
	}
	for _, char := range value {
		if char < 0x21 || char > 0x7e {
			return fmt.Errorf("%w: Idempotency-Key must contain visible ASCII characters", domain.ErrInvalidArgument)
		}
	}
	return nil
}

func returnCaseCode(id uuid.UUID) string {
	compact := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))
	return "RET-" + compact[:12]
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}
