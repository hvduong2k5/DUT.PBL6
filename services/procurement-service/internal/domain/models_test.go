package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func TestNewSupplierNormalizesAndValidates(t *testing.T) {
	supplier, err := domain.NewSupplier(" htx-me ", " Hợp tác xã mè ", "cooperative", " Liên hệ ", "0123", "TEST@EXAMPLE.COM", "Huế")
	if err != nil {
		t.Fatalf("NewSupplier() error = %v", err)
	}
	if supplier.Code != "HTX-ME" || supplier.SupplierType != "COOPERATIVE" || supplier.Email != "test@example.com" || supplier.Status != domain.SupplierActive {
		t.Fatalf("supplier normalization/status unexpected: %+v", supplier)
	}
	if _, err := domain.NewSupplier("", "Name", "COOPERATIVE", "", "", "", ""); err == nil {
		t.Fatal("expected blank supplier code to be rejected")
	}
	if _, err := domain.NewSupplier("SUP-1", "Name", "OTHER", "", "", "", ""); err == nil {
		t.Fatal("expected unsupported supplier type to be rejected")
	}
}

func TestNewRawMaterialNormalizesFields(t *testing.T) {
	material, err := domain.NewRawMaterial(" me-trang ", " Mè trắng ", " kg ", " nguyên liệu ")
	if err != nil {
		t.Fatalf("NewRawMaterial() error = %v", err)
	}
	if material.Code != "ME-TRANG" || material.BaseUnit != "KG" || material.Status != domain.MaterialActive {
		t.Fatalf("material normalization/status unexpected: %+v", material)
	}
}

func TestNewSupplierMaterialOfferValidation(t *testing.T) {
	supplierID, materialID := uuid.New(), uuid.New()
	offer, err := domain.NewSupplierMaterialOffer(supplierID, materialID, "", "kg", 125000, "vnd", "0.25", 3)
	if err != nil {
		t.Fatalf("NewSupplierMaterialOffer() error = %v", err)
	}
	if offer.Unit != "KG" || offer.Currency != "VND" || offer.MinimumQty != "0.25" || !offer.Active {
		t.Fatalf("normalized offer unexpected: %+v", offer)
	}
	tests := []struct {
		name      string
		unitPrice int64
		minQty    string
		leadDays  int
	}{
		{name: "zero price", unitPrice: 0, minQty: "1", leadDays: 0},
		{name: "fraction syntax not accepted by numeric", unitPrice: 10, minQty: "1/2", leadDays: 0},
		{name: "too much scale", unitPrice: 10, minQty: "0.0000001", leadDays: 0},
		{name: "invalid currency", unitPrice: 10, minQty: "1", leadDays: 0},
		{name: "negative lead time", unitPrice: 10, minQty: "1", leadDays: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currency := "VND"
			if tc.name == "invalid currency" {
				currency = "123"
			}
			if _, err := domain.NewSupplierMaterialOffer(supplierID, materialID, "", "KG", tc.unitPrice, currency, tc.minQty, tc.leadDays); err == nil {
				t.Fatal("expected invalid offer to be rejected")
			}
		})
	}
}
