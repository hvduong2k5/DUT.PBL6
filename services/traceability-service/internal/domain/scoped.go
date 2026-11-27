package domain

import "strings"

type BatchContext struct {
	Binding
	WarehouseID    string `json:"warehouse_id"`
	BatchCode      string `json:"batch_code"`
	ProductionDate string `json:"production_date"`
	ExpiryDate     string `json:"expiry_date"`
	Restricted     bool   `json:"restricted"`
	Depleted       bool   `json:"depleted"`
	CatalogVersion string `json:"catalog_version"`
	BatchVersion   string `json:"batch_version"`
}

func ContextFrom(s DependencySnapshot) BatchContext {
	return BatchContext{Binding: s.Binding, WarehouseID: s.WarehouseID, BatchCode: s.BatchCode, ProductionDate: s.ProductionDate, ExpiryDate: s.ExpiryDate, Restricted: s.Restricted, Depleted: s.Depleted, CatalogVersion: s.CatalogVersion, BatchVersion: s.BatchVersion}
}

func Warehouse(p Profile) string {
	if p.WarehouseID == nil {
		return ""
	}
	return *p.WarehouseID
}

func CheckWarehouse(profile Profile, snapshot DependencySnapshot) error {
	if !ValidUUID(snapshot.WarehouseID) {
		return ErrUnavailable
	}
	if profile.WarehouseID == nil || !strings.EqualFold(*profile.WarehouseID, snapshot.WarehouseID) {
		return ErrWarehouseConflict
	}
	return nil
}

type LinkBinding struct {
	SKUCode         string `json:"sku_code"`
	SourceNamespace string `json:"source_namespace"`
	BatchID         string `json:"batch_id"`
}

func (b LinkBinding) Valid() bool {
	return b.SourceNamespace == "inventory" && ValidUUID(b.BatchID) && b.BatchID == strings.ToLower(b.BatchID) && textWithin(b.SKUCode, 100, true) && strings.TrimSpace(b.SKUCode) == b.SKUCode
}

type ResolvedLink struct {
	LinkBinding
	Status string  `json:"status"`
	QRURL  *string `json:"qr_url"`
}
type ResolvedLinks struct {
	Items []ResolvedLink `json:"items"`
	Total int            `json:"total"`
}
type ResolutionProfile struct {
	Profile     *Profile
	Unavailable bool
}
