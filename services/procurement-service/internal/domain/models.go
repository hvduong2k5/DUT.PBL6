package domain

import (
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SupplierStatus string

const (
	SupplierActive    SupplierStatus = "ACTIVE"
	SupplierSuspended SupplierStatus = "SUSPENDED"
)

type Supplier struct {
	ID           uuid.UUID      `json:"id"`
	Code         string         `json:"code"`
	Name         string         `json:"name"`
	SupplierType string         `json:"supplier_type"`
	ContactName  string         `json:"contact_name,omitempty"`
	Phone        string         `json:"phone,omitempty"`
	Email        string         `json:"email,omitempty"`
	Address      string         `json:"address,omitempty"`
	Status       SupplierStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func NewSupplier(code, name, supplierType, contactName, phone, email, address string) (Supplier, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	supplierType = strings.ToUpper(strings.TrimSpace(supplierType))
	if code == "" || name == "" || !validSupplierType(supplierType) {
		return Supplier{}, ErrInvalidArgument
	}
	return Supplier{
		ID: uuid.New(), Code: code, Name: name, SupplierType: supplierType,
		ContactName: strings.TrimSpace(contactName), Phone: strings.TrimSpace(phone),
		Email: strings.ToLower(strings.TrimSpace(email)), Address: strings.TrimSpace(address),
		Status: SupplierActive,
	}, nil
}

func validSupplierType(value string) bool {
	switch value {
	case "FARMER", "COOPERATIVE", "OCOP_BUSINESS", "COMPANY":
		return true
	default:
		return false
	}
}

type MaterialStatus string

const (
	MaterialActive   MaterialStatus = "ACTIVE"
	MaterialInactive MaterialStatus = "INACTIVE"
)

type RawMaterial struct {
	ID          uuid.UUID      `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	BaseUnit    string         `json:"base_unit"`
	Description string         `json:"description,omitempty"`
	Status      MaterialStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func NewRawMaterial(code, name, baseUnit, description string) (RawMaterial, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	baseUnit = strings.ToUpper(strings.TrimSpace(baseUnit))
	if code == "" || name == "" || baseUnit == "" {
		return RawMaterial{}, ErrInvalidArgument
	}
	return RawMaterial{
		ID: uuid.New(), Code: code, Name: name, BaseUnit: baseUnit,
		Description: strings.TrimSpace(description), Status: MaterialActive,
	}, nil
}

type SupplierMaterialOffer struct {
	ID             uuid.UUID `json:"id"`
	SupplierID     uuid.UUID `json:"supplier_id"`
	MaterialID     uuid.UUID `json:"material_id"`
	SupplierSKU    string    `json:"supplier_sku,omitempty"`
	Unit           string    `json:"unit"`
	UnitPriceMinor int64     `json:"unit_price_minor"`
	Currency       string    `json:"currency"`
	MinimumQty     string    `json:"minimum_quantity"`
	LeadTimeDays   int       `json:"lead_time_days"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

var decimalQuantityPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,6})?$`)

func NewSupplierMaterialOffer(supplierID, materialID uuid.UUID, supplierSKU, unit string, unitPriceMinor int64, currency, minimumQty string, leadTimeDays int) (SupplierMaterialOffer, error) {
	unit = strings.ToUpper(strings.TrimSpace(unit))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	minimumQty = strings.TrimSpace(minimumQty)
	quantity, ok := new(big.Rat).SetString(minimumQty)
	if supplierID == uuid.Nil || materialID == uuid.Nil || unit == "" || unitPriceMinor <= 0 || len(currency) != 3 || !ok || quantity.Sign() <= 0 || leadTimeDays < 0 {
		return SupplierMaterialOffer{}, ErrInvalidArgument
	}
	if !decimalQuantityPattern.MatchString(minimumQty) {
		return SupplierMaterialOffer{}, ErrInvalidArgument
	}
	return SupplierMaterialOffer{
		ID: uuid.New(), SupplierID: supplierID, MaterialID: materialID,
		SupplierSKU: strings.TrimSpace(supplierSKU), Unit: unit, UnitPriceMinor: unitPriceMinor,
		Currency: currency, MinimumQty: minimumQty, LeadTimeDays: leadTimeDays, Active: true,
	}, nil
}
