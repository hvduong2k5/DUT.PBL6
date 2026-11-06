package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsAreOrderedAndIncludeSupplierManagement(t *testing.T) {
	entries, err := files.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	if len(names) != 4 || names[0] != "000001_supplier_material_master.up.sql" || names[1] != "000002_purchase_orders.up.sql" || names[2] != "000003_goods_receipts.up.sql" || names[3] != "000004_supplier_agreements_and_price_versions.up.sql" {
		t.Fatalf("unexpected embedded migration sequence: %v", names)
	}
	for _, name := range names {
		if _, err := files.ReadFile(name); err != nil {
			t.Errorf("migration %s is not embedded: %v", name, err)
		}
	}
}

func TestSupplierManagementMigrationDefinesContractsAndEffectiveOfferVersions(t *testing.T) {
	sqlBytes, err := files.ReadFile("000004_supplier_agreements_and_price_versions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"legal_identifier", "preferred_supplier_id", "effective_from", "effective_to", "supplier_contracts", "supplier_contract_status_history"} {
		if !strings.Contains(sql, required) {
			t.Errorf("supplier management migration missing %q", required)
		}
	}
}
