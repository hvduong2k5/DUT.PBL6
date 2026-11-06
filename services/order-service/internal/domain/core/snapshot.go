package core

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var skuPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
var e164Pattern = regexp.MustCompile(`^\+84[35789][0-9]{8}$`)

type Owner struct {
	Kind           string  `json:"kind"`
	PrincipalID    string  `json:"principal_id"`
	CustomerID     *string `json:"customer_id"`
	GuestSessionID *string `json:"guest_session_id"`
}

func (o Owner) Validate() error {
	if !validID(o.PrincipalID) {
		return ErrIdentity
	}
	switch o.Kind {
	case "CUSTOMER":
		if o.CustomerID == nil || !validID(*o.CustomerID) || o.GuestSessionID != nil {
			return ErrIdentity
		}
	case "GUEST":
		if o.GuestSessionID == nil || !validID(*o.GuestSessionID) || o.CustomerID != nil {
			return ErrIdentity
		}
	default:
		return ErrIdentity
	}
	return nil
}

// SameOwner consumes trusted identity mappings. Caller-supplied customer/session
// IDs or a Payment read grant are not credentials for invoking owner commands.
func (o Owner) SameOwner(other Owner) bool {
	if o.Validate() != nil || other.Validate() != nil || o.Kind != other.Kind {
		return false
	}
	if o.Kind == "CUSTOMER" {
		return *o.CustomerID == *other.CustomerID
	}
	return *o.GuestSessionID == *other.GuestSessionID
}

type Provenance struct {
	QuoteID         string        `json:"quote_id"`
	CartID          string        `json:"cart_id"`
	CartRevision    int64         `json:"cart_revision"`
	SelectionDigest SelectionHash `json:"selection_digest"`
	ItemsDigest     ManifestHash  `json:"items_digest"`
	QuoteHash       PayloadHash   `json:"quote_hash"`
	AcceptedAt      time.Time     `json:"accepted_at"`
	CanonicalizedAt time.Time     `json:"canonicalized_at"`
}
type CatalogEvidence struct {
	SnapshotID string    `json:"snapshot_id"`
	Version    string    `json:"version"`
	ValidUntil time.Time `json:"valid_until"`
}
type Packaging struct {
	Description string `json:"description"`
	UnitCode    string `json:"unit_code"`
	Version     string `json:"packaging_version"`
}
type SnapshotItem struct {
	LineID         string    `json:"line_id"`
	SKU            string    `json:"sku_code"`
	Name           string    `json:"product_name"`
	Quantity       int64     `json:"quantity"`
	UnitPrice      int64     `json:"unit_price"`
	LineSubtotal   int64     `json:"line_subtotal"`
	WeightGrams    int64     `json:"weight_grams"`
	Packaging      Packaging `json:"packaging"`
	CatalogVersion string    `json:"catalog_version"`
}
type AddressSource struct {
	Kind                string  `json:"kind"`
	ProfileAddressID    *string `json:"profile_address_id"`
	AddressVersion      *int64  `json:"address_version"`
	ValidationReference string  `json:"validation_reference"`
}
type ShippingAddress struct {
	RecipientName string        `json:"recipient_name"`
	PhoneE164     string        `json:"phone_e164"`
	Street        string        `json:"street"`
	ProvinceCode  string        `json:"province_code"`
	ProvinceName  string        `json:"province_name"`
	WardCode      string        `json:"ward_code"`
	WardName      string        `json:"ward_name"`
	PostalCode    *string       `json:"postal_code"`
	Source        AddressSource `json:"source"`
}
type ShippingTerms struct {
	CarrierCode        string    `json:"carrier_code"`
	ServiceCode        string    `json:"service_code"`
	QuoteReference     string    `json:"quote_reference"`
	QuoteVersion       string    `json:"quote_version"`
	ValidUntil         time.Time `json:"valid_until"`
	PackageWeightGrams int64     `json:"package_weight_grams"`
	QuotedFee          int64     `json:"quoted_fee"`
}
type SnapshotMoney struct {
	Currency            string `json:"currency"`
	Subtotal            int64  `json:"subtotal"`
	MerchandiseDiscount int64  `json:"merchandise_discount"`
	ShippingFee         int64  `json:"shipping_fee"`
	ShippingDiscount    int64  `json:"shipping_discount"`
	FinalAmount         int64  `json:"final_amount"`
}
type PaymentTerms struct {
	Method                 string     `json:"method"`
	Provider               *string    `json:"provider"`
	ExpiresAt              *time.Time `json:"payment_expires_at"`
	ReceiverProfileID      *string    `json:"receiver_profile_id"`
	ReceiverProfileVersion *int64     `json:"receiver_profile_version"`
	CODPolicyReference     *string    `json:"cod_policy_reference"`
	CollectAmount          *int64     `json:"collect_amount"`
}
type Policies struct {
	Order     string  `json:"order_policy_version"`
	Checkout  string  `json:"checkout_policy_version"`
	Payment   string  `json:"payment_policy_version"`
	Shipping  string  `json:"shipping_policy_version"`
	Promotion *string `json:"promotion_policy_version"`
}
type SnapshotInput struct {
	SchemaVersion string          `json:"snapshot_schema_version"`
	SnapshotID    string          `json:"snapshot_id"`
	OrderID       string          `json:"order_id"`
	OperationID   string          `json:"operation_id"`
	MerchantID    string          `json:"merchant_id"`
	WarehouseID   string          `json:"warehouse_id"`
	Channel       string          `json:"channel"`
	PlacedAt      time.Time       `json:"placed_at"`
	Owner         Owner           `json:"owner_at_placement"`
	Provenance    Provenance      `json:"provenance"`
	Catalog       CatalogEvidence `json:"catalog"`
	Items         []SnapshotItem  `json:"items"`
	Address       ShippingAddress `json:"shipping_address"`
	Shipping      ShippingTerms   `json:"shipping"`
	Money         SnapshotMoney   `json:"money"`
	Payment       PaymentTerms    `json:"payment_terms"`
	Policies      Policies        `json:"policies"`
}

// Snapshot stores only immutable canonical bytes. Input slices/pointers and data
// returned to callers cannot mutate the commercial document or its logical hash.
type Snapshot struct {
	canonical string
	hash      PayloadHash
}

func NewSnapshot(input SnapshotInput) (Snapshot, error) {
	if !validInputStrings(reflect.ValueOf(input)) {
		return Snapshot{}, ErrSnapshot
	}
	b, err := json.Marshal(input)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	return DecodeSnapshot(b)
}

func DecodeSnapshot(data []byte) (Snapshot, error) {
	v, err := parseJSON(data)
	if err != nil || requireShape(v, reflect.TypeOf(SnapshotInput{})) != nil {
		return Snapshot{}, ErrSnapshot
	}
	var input SnapshotInput
	if json.Unmarshal(data, &input) != nil || validateSnapshot(input) != nil {
		return Snapshot{}, ErrSnapshot
	}
	sort.Slice(input.Items, func(i, j int) bool { return input.Items[i].SKU < input.Items[j].SKU })
	b, err := canonicalJSON(input)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	return Snapshot{canonical: string(b), hash: PayloadHash(digest(b))}, nil
}

func (s Snapshot) Bytes() []byte     { return []byte(s.canonical) }
func (s Snapshot) Hash() PayloadHash { return s.hash }
func (s Snapshot) Data() SnapshotInput {
	var input SnapshotInput
	_ = json.Unmarshal([]byte(s.canonical), &input) // Constructor owns these bytes.
	return input
}

func text(s string) bool { return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= 255 }

func validateSnapshot(s SnapshotInput) error {
	if s.SchemaVersion != "1.0" || s.Channel != "D2C_WEB" || s.Owner.Validate() != nil ||
		!validID(s.SnapshotID) || !validID(s.OrderID) || !validID(s.OperationID) || !validID(s.MerchantID) ||
		!text(s.WarehouseID) || s.PlacedAt.IsZero() {
		return ErrSnapshot
	}
	p := s.Provenance
	if !validID(p.QuoteID) || !validID(p.CartID) || p.CartRevision < 0 || !validHash(string(p.QuoteHash)) ||
		p.AcceptedAt.IsZero() || p.CanonicalizedAt.Before(p.AcceptedAt) || s.PlacedAt.Before(p.CanonicalizedAt) {
		return ErrSnapshot
	}
	if !text(s.Catalog.SnapshotID) || !text(s.Catalog.Version) || !s.PlacedAt.Before(s.Catalog.ValidUntil) ||
		!s.PlacedAt.Before(s.Shipping.ValidUntil) || len(s.Items) == 0 || len(s.Items) > 100 {
		return ErrSnapshot
	}
	var lines []SelectionLine
	seen := map[string]bool{}
	var subtotal Money
	var weight int64
	for _, item := range s.Items {
		if !validID(item.LineID) || seen[item.LineID] || !text(item.Name) || item.WeightGrams < 1 || item.WeightGrams > 1_000_000_000 ||
			!text(item.Packaging.Description) || !text(item.Packaging.UnitCode) || !text(item.Packaging.Version) ||
			item.CatalogVersion != s.Catalog.Version || item.UnitPrice < 1 {
			return ErrSnapshot
		}
		seen[item.LineID] = true
		price, err := NewMoney("VND", item.UnitPrice)
		if err != nil {
			return ErrSnapshot
		}
		line, err := price.Multiply(item.Quantity)
		if err != nil || line.Amount() != item.LineSubtotal {
			return ErrSnapshot
		}
		subtotal, err = subtotal.Add(line)
		if err != nil {
			return ErrSnapshot
		}
		if item.Quantity < 1 || item.Quantity > 999 || weight > 1_000_000_000-item.WeightGrams*item.Quantity {
			return ErrSnapshot
		}
		weight += item.WeightGrams * item.Quantity
		lines = append(lines, SelectionLine{Quantity: item.Quantity, SKU: item.SKU})
	}
	sel, err := HashSelection(lines)
	manifest, manifestErr := HashManifest(lines)
	if err != nil || manifestErr != nil || sel != p.SelectionDigest || manifest != p.ItemsDigest {
		return ErrSnapshot
	}
	m := s.Money
	fee, err := NewMoney(m.Currency, m.ShippingFee)
	if err != nil {
		return ErrSnapshot
	}
	final, err := subtotal.Add(fee)
	if err != nil || m.MerchandiseDiscount != 0 || m.ShippingDiscount != 0 || m.Subtotal != subtotal.Amount() ||
		m.FinalAmount < 1 || final.Amount() != m.FinalAmount || s.Shipping.QuotedFee != m.ShippingFee ||
		s.Shipping.PackageWeightGrams < weight || s.Shipping.PackageWeightGrams > 1_000_000_000 {
		return ErrSnapshot
	}
	for _, value := range []string{s.Shipping.CarrierCode, s.Shipping.ServiceCode, s.Shipping.QuoteReference, s.Shipping.QuoteVersion,
		s.Address.RecipientName, s.Address.Street, s.Address.ProvinceCode, s.Address.ProvinceName, s.Address.WardCode, s.Address.WardName,
		s.Address.Source.ValidationReference, s.Policies.Order, s.Policies.Checkout, s.Policies.Payment, s.Policies.Shipping} {
		if !text(value) {
			return ErrSnapshot
		}
	}
	if !e164Pattern.MatchString(s.Address.PhoneE164) || s.Policies.Promotion != nil ||
		(s.Address.PostalCode != nil && !text(*s.Address.PostalCode)) {
		return ErrSnapshot
	}
	a := s.Address.Source
	switch a.Kind {
	case "PROFILE_ADDRESS":
		if a.ProfileAddressID == nil || !validID(*a.ProfileAddressID) || a.AddressVersion == nil || *a.AddressVersion < 1 || s.Owner.Kind != "CUSTOMER" {
			return ErrSnapshot
		}
	case "VALIDATED_INLINE":
		if a.ProfileAddressID != nil || a.AddressVersion != nil {
			return ErrSnapshot
		}
	default:
		return ErrSnapshot
	}
	t := s.Payment
	switch t.Method {
	case "VIETQR":
		if t.Provider == nil || *t.Provider != "SEPAY" || t.ExpiresAt == nil || !s.PlacedAt.Before(*t.ExpiresAt) ||
			t.ReceiverProfileID == nil || !validID(*t.ReceiverProfileID) || t.ReceiverProfileVersion == nil || *t.ReceiverProfileVersion < 1 ||
			t.CODPolicyReference != nil || t.CollectAmount != nil {
			return ErrSnapshot
		}
	case "COD":
		if t.Provider != nil || t.ExpiresAt != nil || t.ReceiverProfileID != nil || t.ReceiverProfileVersion != nil ||
			t.CODPolicyReference == nil || !text(*t.CODPolicyReference) || t.CollectAmount == nil || *t.CollectAmount != m.FinalAmount {
			return ErrSnapshot
		}
	default:
		return ErrSnapshot
	}
	return nil
}
