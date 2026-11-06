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
	if len(names) != 7 || names[0] != "000001_supplier_material_master.up.sql" || names[1] != "000002_purchase_orders.up.sql" || names[2] != "000003_goods_receipts.up.sql" || names[3] != "000004_supplier_agreements_and_price_versions.up.sql" || names[4] != "000005_purchase_order_supplier_response_and_revisions.up.sql" || names[5] != "000006_goods_receipt_idempotency.up.sql" || names[6] != "000007_goods_receipt_variances.up.sql" {
		t.Fatalf("unexpected embedded migration sequence: %v", names)
	}
	for _, name := range names {
		if _, err := files.ReadFile(name); err != nil {
			t.Errorf("migration %s is not embedded: %v", name, err)
		}
	}
}

func TestGoodsReceiptVarianceMigrationAddsJSONArray(t *testing.T) {
	sqlBytes, err := files.ReadFile("000007_goods_receipt_variances.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"variances JSONB", "DEFAULT '[]'::jsonb", "jsonb_typeof(variances) = 'array'"} {
		if !strings.Contains(sql, required) {
			t.Errorf("goods receipt variance migration missing %q", required)
		}
	}
}

func TestGoodsReceiptIdempotencyMigrationAddsUniqueActorKeyAndPayloadHash(t *testing.T) {
	sqlBytes, err := files.ReadFile("000006_goods_receipt_idempotency.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"idempotency_key", "idempotency_request_hash", "UNIQUE INDEX", "goods_receipts_received_by_idempotency_key_unique"} {
		if !strings.Contains(sql, required) {
			t.Errorf("goods receipt idempotency migration missing %q", required)
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

func TestPurchaseOrderLifecycleMigrationIncludesResponseRevisionAndCloseStates(t *testing.T) {
	sqlBytes, err := files.ReadFile("000005_purchase_order_supplier_response_and_revisions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"ACKNOWLEDGED", "CHANGE_REQUESTED", "SUPPLIER_DECLINED", "CLOSED", "supplier_responded_at", "closed_at", "purchase_order_revision_snapshots"} {
		if !strings.Contains(sql, required) {
			t.Errorf("PO lifecycle migration missing %q", required)
		}
	}
}
