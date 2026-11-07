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
	ID              uuid.UUID      `json:"id"`
	Code            string         `json:"code"`
	Name            string         `json:"name"`
	SupplierType    string         `json:"supplier_type"`
	ContactName     string         `json:"contact_name,omitempty"`
	Phone           string         `json:"phone,omitempty"`
	Email           string         `json:"email,omitempty"`
	Address         string         `json:"address,omitempty"`
	LegalIdentifier string         `json:"legal_identifier,omitempty"`
	Status          SupplierStatus `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func NewSupplier(code, name, supplierType, contactName, phone, email, address string) (Supplier, error) {
	return NewSupplierWithLegalIdentifier(code, name, supplierType, contactName, phone, email, address, "")
}

func NewSupplierWithLegalIdentifier(code, name, supplierType, contactName, phone, email, address, legalIdentifier string) (Supplier, error) {
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
		LegalIdentifier: strings.ToUpper(strings.TrimSpace(legalIdentifier)),
		Status:          SupplierActive,
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
	ID                  uuid.UUID      `json:"id"`
	Code                string         `json:"code"`
	Name                string         `json:"name"`
	BaseUnit            string         `json:"base_unit"`
	Description         string         `json:"description,omitempty"`
	Status              MaterialStatus `json:"status"`
	PreferredSupplierID *uuid.UUID     `json:"preferred_supplier_id,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

func (m *RawMaterial) Replace(updated RawMaterial) error {
	if m.ID == uuid.Nil || updated.Code == "" || updated.Name == "" || updated.BaseUnit == "" {
		return ErrInvalidArgument
	}
	updated.ID = m.ID
	updated.Status = m.Status
	updated.CreatedAt = m.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	*m = updated
	return nil
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
	ID             uuid.UUID  `json:"id"`
	SupplierID     uuid.UUID  `json:"supplier_id"`
	MaterialID     uuid.UUID  `json:"material_id"`
	SupplierSKU    string     `json:"supplier_sku,omitempty"`
	Unit           string     `json:"unit"`
	UnitPriceMinor int64      `json:"unit_price_minor"`
	Currency       string     `json:"currency"`
	MinimumQty     string     `json:"minimum_quantity"`
	LeadTimeDays   int        `json:"lead_time_days"`
	Active         bool       `json:"active"`
	Version        int        `json:"version"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (o SupplierMaterialOffer) EffectiveAt(at time.Time) bool {
	at = at.UTC()
	return o.Active && !o.EffectiveFrom.After(at) && (o.EffectiveTo == nil || at.Before(*o.EffectiveTo))
}

var (
	decimalQuantityPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,6})?$`)
	currencyPattern        = regexp.MustCompile(`^[A-Z]{3}$`)
)

func NewSupplierMaterialOffer(supplierID, materialID uuid.UUID, supplierSKU, unit string, unitPriceMinor int64, currency, minimumQty string, leadTimeDays int) (SupplierMaterialOffer, error) {
	return NewSupplierMaterialOfferVersion(supplierID, materialID, supplierSKU, unit, unitPriceMinor, currency, minimumQty, leadTimeDays, time.Now().UTC(), nil)
}

func NewSupplierMaterialOfferVersion(supplierID, materialID uuid.UUID, supplierSKU, unit string, unitPriceMinor int64, currency, minimumQty string, leadTimeDays int, effectiveFrom time.Time, effectiveTo *time.Time) (SupplierMaterialOffer, error) {
	unit = strings.ToUpper(strings.TrimSpace(unit))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	minimumQty = strings.TrimSpace(minimumQty)
	quantity, ok := new(big.Rat).SetString(minimumQty)
	effectiveFrom = effectiveFrom.UTC()
	if effectiveFrom.IsZero() {
		effectiveFrom = time.Now().UTC()
	}
	if effectiveTo != nil {
		end := effectiveTo.UTC()
		if !effectiveFrom.Before(end) {
			return SupplierMaterialOffer{}, ErrInvalidArgument
		}
		effectiveTo = &end
	}
	if supplierID == uuid.Nil || materialID == uuid.Nil || unit == "" || unitPriceMinor <= 0 || !currencyPattern.MatchString(currency) || !ok || quantity.Sign() <= 0 || leadTimeDays < 0 {
		return SupplierMaterialOffer{}, ErrInvalidArgument
	}
	if !decimalQuantityPattern.MatchString(minimumQty) {
		return SupplierMaterialOffer{}, ErrInvalidArgument
	}
	return SupplierMaterialOffer{
		ID: uuid.New(), SupplierID: supplierID, MaterialID: materialID,
		SupplierSKU: strings.TrimSpace(supplierSKU), Unit: unit, UnitPriceMinor: unitPriceMinor,
		Currency: currency, MinimumQty: minimumQty, LeadTimeDays: leadTimeDays, Active: true,
		Version: 1, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo,
	}, nil
}

type SupplierContractStatus string

const (
	SupplierContractDraft      SupplierContractStatus = "DRAFT"
	SupplierContractActive     SupplierContractStatus = "ACTIVE"
	SupplierContractExpired    SupplierContractStatus = "EXPIRED"
	SupplierContractTerminated SupplierContractStatus = "TERMINATED"
)

type SupplierContract struct {
	ID             uuid.UUID              `json:"id"`
	SupplierID     uuid.UUID              `json:"supplier_id"`
	ContractNumber string                 `json:"contract_number"`
	Revision       int                    `json:"revision"`
	Scope          string                 `json:"scope"`
	PaymentTerms   string                 `json:"payment_terms"`
	DocumentFileID string                 `json:"document_file_id,omitempty"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	EffectiveTo    *time.Time             `json:"effective_to,omitempty"`
	Status         SupplierContractStatus `json:"status"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func NewSupplierContract(supplierID uuid.UUID, number, scope, paymentTerms, documentFileID string, from time.Time, to *time.Time, actor string) (SupplierContract, error) {
	number, scope, paymentTerms, documentFileID, actor = strings.TrimSpace(number), strings.TrimSpace(scope), strings.TrimSpace(paymentTerms), strings.TrimSpace(documentFileID), strings.TrimSpace(actor)
	from = from.UTC()
	if supplierID == uuid.Nil || number == "" || scope == "" || paymentTerms == "" || actor == "" || from.IsZero() {
		return SupplierContract{}, ErrInvalidArgument
	}
	if to != nil {
		end := to.UTC()
		if !from.Before(end) {
			return SupplierContract{}, ErrInvalidArgument
		}
		to = &end
	}
	now := time.Now().UTC()
	return SupplierContract{ID: uuid.New(), SupplierID: supplierID, ContractNumber: number, Revision: 1, Scope: scope, PaymentTerms: paymentTerms, DocumentFileID: documentFileID, EffectiveFrom: from, EffectiveTo: to, Status: SupplierContractDraft, CreatedBy: actor, CreatedAt: now, UpdatedAt: now}, nil
}
