package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxMoney int64 = 1_000_000_000_000

var ErrConflict = errors.New("conflict")

type Error struct {
	Code string `json:"code"`
	HTTP int    `json:"-"`
}

func (e *Error) Error() string        { return e.Code }
func E(code string, status int) error { return &Error{code, status} }

type Principal struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Warehouse  string `json:"warehouse"`
	CustomerID string `json:"customer_id,omitempty"`
}

func (p Principal) Scope() string { return strings.ToLower(p.Role) + ":" + p.ID }
func (p Principal) Staff() bool {
	return p.Role == "ADMIN" || p.Role == "SALES_MANAGER" || p.Role == "WAREHOUSE_STAFF" || p.Role == "PACKING_STAFF" || p.Role == "CUSTOMER_SERVICE"
}
func (p Principal) Manager() bool { return p.Role == "ADMIN" || p.Role == "SALES_MANAGER" }

type Item struct {
	ItemID    string `json:"item_id,omitempty"`
	SKU       string `json:"sku_code"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price,omitempty"`
	Name      string `json:"product_name,omitempty"`
	Weight    int64  `json:"weight_grams,omitempty"`
	Version   int64  `json:"catalog_version,omitempty"`
}
type Address struct {
	Name     string `json:"recipient_name"`
	Phone    string `json:"phone"`
	Street   string `json:"street"`
	Ward     string `json:"ward_code"`
	Province string `json:"province_code"`
}

var phonePattern = regexp.MustCompile(`^(?:\+84|0)[35789][0-9]{8}$`)
var skuPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func (a Address) Validate() error {
	for _, v := range []string{a.Name, a.Phone, a.Street, a.Ward, a.Province} {
		if len(strings.TrimSpace(v)) == 0 || len(v) > 255 {
			return E("INVALID_ADDRESS", 422)
		}
	}
	if !phonePattern.MatchString(a.Phone) {
		return E("INVALID_ADDRESS", 422)
	}
	return nil
}

type Cart struct {
	Revision int64  `json:"revision"`
	Items    []Item `json:"items"`
}
type QuoteInput struct {
	Revision  int64    `json:"cart_revision"`
	Method    string   `json:"payment_method"`
	AddressID string   `json:"shipping_address_id,omitempty"`
	Address   *Address `json:"address,omitempty"`
	Voucher   string   `json:"voucher_code,omitempty"`
}
type CheckoutInput struct {
	QuoteID  string `json:"quote_id"`
	Revision int64  `json:"cart_revision"`
	Method   string `json:"payment_method"`
}
type Quote struct {
	CatalogSnapshotID string    `json:"catalog_snapshot_id"`
	CatalogExpiresAt  time.Time `json:"catalog_expires_at"`
	ShippingReference string    `json:"shipping_quote_reference"`
	AddressID         string    `json:"source_address_id,omitempty"`
	AddressVersion    int64     `json:"source_address_version,omitempty"`
	ID                string    `json:"quote_id"`
	Scope             string    `json:"-"`
	Principal         Principal `json:"-"`
	Revision          int64     `json:"cart_revision"`
	Method            string    `json:"payment_method"`
	Items             []Item    `json:"items"`
	Address           Address   `json:"address"`
	Subtotal          int64     `json:"subtotal"`
	Shipping          int64     `json:"shipping_fee"`
	Final             int64     `json:"final_amount"`
	CustomerID        string    `json:"customer_id,omitempty"`
	ExpiresAt         time.Time `json:"expires_at"`
	ShippingExpiresAt time.Time `json:"shipping_expires_at"`
}

// StoredQuote retains server ownership; Quote's public JSON intentionally omits scope/principal.
type StoredQuote struct {
	Quote     Quote     `json:"quote"`
	Scope     string    `json:"scope"`
	Principal Principal `json:"principal"`
}
type Operation struct {
	ID          string    `json:"operation_id"`
	Status      string    `json:"status"`
	OrderID     *string   `json:"order_id"`
	OrderStatus string    `json:"order_status,omitempty"`
	ErrorCode   string    `json:"error_code,omitempty"`
	AcceptedAt  time.Time `json:"accepted_at"`
}
type Order struct {
	ID            string     `json:"order_id"`
	OperationID   string     `json:"operation_id"`
	Code          string     `json:"order_code"`
	Scope         string     `json:"-"`
	PrincipalID   string     `json:"-"`
	CustomerID    *string    `json:"customer_id"`
	Warehouse     string     `json:"warehouse_id"`
	Method        string     `json:"payment_method"`
	Status        string     `json:"status"`
	Payment       string     `json:"payment_status"`
	Stock         string     `json:"stock_status"`
	Subtotal      int64      `json:"subtotal"`
	Shipping      int64      `json:"shipping_fee"`
	Final         int64      `json:"final_amount"`
	Version       int64      `json:"version"`
	Generation    int64      `json:"ready_generation"`
	Ready         bool       `json:"ready_authorized"`
	Hold          bool       `json:"operational_hold"`
	ActiveCase    bool       `json:"has_active_case"`
	Reservation   string     `json:"-"`
	Reference     string     `json:"payment_reference"`
	Task          *string    `json:"task_id"`
	Shipment      *string    `json:"shipment_id"`
	Seal          *string    `json:"seal_code"`
	ExpiresAt     time.Time  `json:"payment_expires_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatedAt     time.Time  `json:"created_at"`
	SLA           *time.Time `json:"sla_due_at"`
	CompletionDue *time.Time `json:"completion_due_at"`
	Snapshot      *Quote     `json:"snapshot,omitempty"`
}
type Receipt struct {
	Provider      string    `json:"provider"`
	TransactionID string    `json:"transaction_id"`
	Reference     string    `json:"reference"`
	Receiver      string    `json:"receiver"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaidAt        time.Time `json:"paid_at"`
}

func (r Receipt) Validate(receiver string) error {
	if r.Provider != "SIMULATOR" || r.Status != "SUCCEEDED" || r.Receiver != receiver || r.Currency != "VND" || r.Amount <= 0 || r.Amount > MaxMoney || len(r.TransactionID) == 0 || len(r.TransactionID) > 128 || len(r.Reference) > 128 || r.PaidAt.IsZero() {
		return E("INVALID_RECEIPT", 422)
	}
	return nil
}

type Event struct {
	Signature  string `json:"signature"`
	ID         string `json:"id"`
	Source     string `json:"source"`
	OrderID    string `json:"order_id"`
	ResourceID string `json:"resource_id"`
	Version    int64  `json:"version"`
	Type       string `json:"type"`
	Generation int64  `json:"generation,omitempty"`
	Seal       string `json:"seal_code,omitempty"`
}
type Reservation struct {
	ID        string    `json:"reservation_id"`
	OrderID   string    `json:"order_id"`
	State     string    `json:"state"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Approval struct {
	ID        string `json:"approval_id"`
	OrderID   string `json:"order_id"`
	ReceiptID string `json:"receipt_id"`
	Amount    int64  `json:"amount"`
	Type      string `json:"type"`
	State     string `json:"state"`
	Actor     string `json:"actor"`
}

func Hash(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func Total(items []Item) (int64, error) {
	if len(items) == 0 || len(items) > 100 {
		return 0, E("INVALID_ITEMS", 422)
	}
	var sum int64
	seen := map[string]bool{}
	for _, i := range items {
		if len(i.SKU) == 0 || len(i.SKU) > 64 || i.Quantity < 1 || i.Quantity > 999 || i.UnitPrice < 1 || i.UnitPrice > MaxMoney/int64(i.Quantity) || seen[i.SKU] {
			return 0, E("INVALID_ITEMS", 422)
		}
		seen[i.SKU] = true
		line := i.UnitPrice * int64(i.Quantity)
		if sum > MaxMoney-line {
			return 0, E("MONEY_OVERFLOW", 422)
		}
		sum += line
	}
	return sum, nil
}
func StateAllowed(from, to string) bool {
	transitions := map[string][]string{
		"PENDING_PAYMENT":    {"PAYMENT_FINALIZING", "CANCELLED_TIMEOUT", "CANCELLED_BY_USER"},
		"PAYMENT_FINALIZING": {"PAID", "CANCELLED_TIMEOUT"},
		"PAID":               {"PROCESSING", "CANCELLED_BY_ADMIN"}, "CONFIRMED_COD": {"PROCESSING", "CANCELLED_BY_USER", "CANCELLED_BY_ADMIN"},
		"PROCESSING": {"PACKED"}, "PACKED": {"SHIPPED"}, "SHIPPED": {"DELIVERED", "DELIVERY_FAILED"}, "DELIVERY_FAILED": {"SHIPPED"}, "DELIVERED": {"COMPLETED"},
	}
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
func RequireTransition(from, to string) error {
	if !StateAllowed(from, to) {
		return E(fmt.Sprintf("INVALID_TRANSITION_%s_TO_%s", from, to), 409)
	}
	return nil
}

func EventSignature(event Event, key string) string {
	event.Signature = ""
	b, _ := json.Marshal(event)
	m := hmac.New(sha256.New, []byte(key))
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}
