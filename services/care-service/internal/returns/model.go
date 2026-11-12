package returns

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
)

// OrderContext is the proposed Order-owned snapshot for Care return intake.
// It is fetched server-to-server; customer requests never supply authoritative
// quantities, eligibility, money ceilings, addresses, or parcel dimensions.
type OrderContext struct {
	OrderID              string      `json:"order_id"`
	OrderNumber          string      `json:"order_number"`
	CustomerID           uuid.UUID   `json:"customer_id"`
	Channel              string      `json:"channel"`
	Status               string      `json:"status"`
	ContractVersion      string      `json:"contract_version"`
	CapturedAt           time.Time   `json:"captured_at"`
	ReturnEligibleAt     time.Time   `json:"return_eligible_at"`
	EligibilityExpiresAt time.Time   `json:"eligibility_expires_at"`
	PickupAddress        Address     `json:"pickup_address"`
	Destination          Address     `json:"destination"`
	Parcel               Parcel      `json:"parcel"`
	Lines                []OrderLine `json:"lines"`
}

type Address struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Street      string `json:"street"`
	Ward        string `json:"ward"`
	District    string `json:"district,omitempty"`
	Province    string `json:"province"`
	CountryCode string `json:"country_code,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
}

type Parcel struct {
	WeightGrams int64 `json:"weight_grams"`
	LengthCM    int64 `json:"length_cm"`
	WidthCM     int64 `json:"width_cm"`
	HeightCM    int64 `json:"height_cm"`
	Fragile     bool  `json:"fragile,omitempty"`
}

type OrderLine struct {
	OrderLineID                       string `json:"order_line_id"`
	ProductID                         string `json:"product_id"`
	SKU                               string `json:"sku"`
	ProductName                       string `json:"product_name"`
	PurchasedQuantity                 int    `json:"purchased_quantity"`
	PreviouslyCommittedReturnQuantity int    `json:"previously_committed_return_quantity"`
	RefundableUnitCeilingMinor        int64  `json:"refundable_unit_ceiling_minor"`
	Currency                          string `json:"currency"`
}

type CreateInput struct {
	ActorID         uuid.UUID
	OrderID         string
	Resolution      string
	ReasonCode      string
	CustomerNote    string
	Lines           []RequestedLine
	EvidenceFileIDs []uuid.UUID
	RequestID       string
	IdempotencyKey  string
}

type RequestedLine struct {
	OrderLineID string `json:"order_line_id"`
	Quantity    int    `json:"quantity"`
}

type CaseLine struct {
	ID                    uuid.UUID `json:"case_line_id"`
	OrderLineID           string    `json:"order_line_id"`
	ProductID             string    `json:"product_id"`
	SKU                   string    `json:"sku"`
	ProductName           string    `json:"product_name"`
	PurchasedQuantity     int       `json:"purchased_quantity"`
	RequestedQuantity     int       `json:"requested_quantity"`
	RefundableCeiling     int64     `json:"refundable_ceiling_minor"`
	RefundableUnitCeiling int64     `json:"refundable_unit_ceiling_minor"`
	Currency              string    `json:"currency"`
}

type Case struct {
	ID                uuid.UUID               `json:"case_id"`
	Code              string                  `json:"case_number"`
	CustomerID        uuid.UUID               `json:"customer_id,omitempty"`
	OrderID           string                  `json:"order_id"`
	OrderNumber       string                  `json:"order_number,omitempty"`
	Status            domain.ReturnCaseStatus `json:"status"`
	Resolution        string                  `json:"requested_resolution"`
	ReasonCode        string                  `json:"reason_code"`
	CustomerNote      string                  `json:"customer_note,omitempty"`
	Version           int                     `json:"revision"`
	OrderContext      OrderContext            `json:"-"`
	Lines             []CaseLine              `json:"lines"`
	EvidenceFileIDs   []uuid.UUID             `json:"evidence_file_ids,omitempty"`
	Decision          *Decision               `json:"decision,omitempty"`
	ReverseShipmentID string                  `json:"reverse_shipment_id,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

type DecisionLineInput struct {
	ReturnCaseLineID    uuid.UUID `json:"return_case_line_id"`
	ApprovedQuantity    int       `json:"approved_quantity"`
	ApprovedAmountMinor int64     `json:"approved_amount_minor"`
	ReasonCode          string    `json:"reason_code"`
}

type Decision struct {
	ID                   uuid.UUID               `json:"decision_id"`
	Version              int                     `json:"decision_version"`
	Outcome              domain.ReturnCaseStatus `json:"outcome"`
	ActorID              uuid.UUID               `json:"decided_by"`
	ReasonCode           string                  `json:"reason_code"`
	Reason               string                  `json:"reason"`
	ReturnRequired       bool                    `json:"return_required"`
	ApprovedTotalMinor   int64                   `json:"approved_total_minor"`
	AuthoritativeCeiling int64                   `json:"authoritative_ceiling_minor"`
	Currency             string                  `json:"currency"`
	Lines                []DecisionLineInput     `json:"lines"`
	DecidedAt            time.Time               `json:"decided_at"`
}

type ActionInput struct {
	ActorID         uuid.UUID
	CaseID          uuid.UUID
	Action          string
	ExpectedVersion int
	ReasonCode      string
	Note            string
	ReturnRequired  bool
	Lines           []DecisionLineInput
	RequestID       string
	IdempotencyKey  string
}

type CreateRecord struct {
	Case              Case
	OrderContext      OrderContext
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
	RequestID         string
}

type ActionRecord struct {
	CaseID            uuid.UUID
	ActorID           uuid.UUID
	ExpectedVersion   int
	Action            string
	ReasonCode        string
	Note              string
	Decision          *Decision
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
	RequestID         string
	At                time.Time
}

type OrderContextProvider interface {
	GetReturnContext(context.Context, string) (OrderContext, error)
}

type Repository interface {
	ReplayCreate(context.Context, uuid.UUID, string, string) (Case, bool, error)
	Create(context.Context, CreateRecord) (Case, bool, error)
	GetOwned(context.Context, uuid.UUID, uuid.UUID) (Case, error)
	ListOwned(context.Context, uuid.UUID, domain.ReturnCaseStatus, int, int) ([]Case, int, error)
	Get(context.Context, uuid.UUID) (Case, error)
	List(context.Context, domain.ReturnCaseStatus, int, int) ([]Case, int, error)
	StatusCounts(context.Context) (map[domain.ReturnCaseStatus]int, error)
	Action(context.Context, ActionRecord) (Case, bool, error)
	AttachReverseShipment(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (Case, error)
}
