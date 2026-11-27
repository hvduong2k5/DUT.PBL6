package migrations

import (
	"os"
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
	want := []string{
		"000001_supplier_material_master.up.sql",
		"000002_purchase_orders.up.sql",
		"000003_goods_receipts.up.sql",
		"000004_supplier_agreements_and_price_versions.up.sql",
		"000005_purchase_order_supplier_response_and_revisions.up.sql",
		"000006_goods_receipt_idempotency.up.sql",
		"000007_goods_receipt_variances.up.sql",
		"000008_supply_plan_snapshots.up.sql",
		"000009_supply_plan_revision_immutability.up.sql",
		"000010_supply_plan_approval_workflow.up.sql",
		"000011_supply_plan_purchase_order_conversion.up.sql",
		"000012_purchase_order_idempotency.up.sql",
		"000013_goods_received_outbox.up.sql",
		"000014_receipt_inspection_idempotency.up.sql",
		"000015_purchase_order_action_idempotency.up.sql",
		"000016_purchase_order_terminal_actions.up.sql",
		"000017_purchase_order_revision_evidence.up.sql",
		"000018_purchase_order_draft_edit_commands.up.sql",
		"000019_receipt_variance_resolution_results.up.sql",
	}
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected embedded migration sequence: %v", names)
	}
	for _, name := range names {
		if _, err := files.ReadFile(name); err != nil {
			t.Errorf("migration %s is not embedded: %v", name, err)
		}
	}
}

func TestPOActionMigrationStoresCompletedSnapshotsWithActorScopedKeys(t *testing.T) {
	encoded, err := files.ReadFile("000015_purchase_order_action_idempotency.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"PRIMARY KEY (actor_id, idempotency_key)", "REFERENCES purchase_orders(id) ON DELETE RESTRICT", "expected_version BIGINT", "result_snapshot JSONB NOT NULL", "request_hash VARCHAR(64)", "'SUBMIT','APPROVE','REJECT','SEND'"} {
		if !strings.Contains(string(encoded), required) {
			t.Fatalf("missing command evidence guard: %s", required)
		}
	}
}

func TestTerminalPOActionsExtendJournalWithoutDiscardingEvidence(t *testing.T) {
	up, err := files.ReadFile("000016_purchase_order_terminal_actions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"'SUBMIT'", "'SEND'", "'ACKNOWLEDGE'", "'REQUEST_CHANGE'", "'DECLINE'", "'CANCEL'", "'CLOSE'"} {
		if !strings.Contains(string(up), action) {
			t.Fatalf("missing journal action: %s", action)
		}
	}
	down, err := os.ReadFile("000016_purchase_order_terminal_actions.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "RAISE EXCEPTION") || strings.Contains(string(down), "DELETE FROM") || strings.Contains(string(up), "DROP TABLE") {
		t.Fatal("rollback must refuse incompatible completed evidence, never discard it")
	}
}

func TestPORevisionMigrationProtectsArchivedEvidence(t *testing.T) {
	up, err := files.ReadFile("000017_purchase_order_revision_evidence.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, guard := range []string{"'REVISION'", "BEFORE UPDATE OR DELETE ON purchase_order_revision_snapshots", "BEFORE TRUNCATE ON purchase_order_revision_snapshots", "RAISE EXCEPTION"} {
		if !strings.Contains(string(up), guard) {
			t.Fatalf("missing archive guard: %s", guard)
		}
	}
	down, err := os.ReadFile("000017_purchase_order_revision_evidence.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "RAISE EXCEPTION") || strings.Contains(string(down), "DELETE FROM") || strings.Contains(string(down), "DROP TABLE") {
		t.Fatal("rollback must retain archives and completed commands")
	}
}

func TestDraftEditMigrationRetainsPriorCommands(t *testing.T) {
	up, err := files.ReadFile("000018_purchase_order_draft_edit_commands.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(up), "'EDIT_DRAFT'") || !strings.Contains(string(up), "'REVISION'") || strings.Contains(string(up), "DROP TABLE") {
		t.Fatal("draft edit migration lost existing journal")
	}
	down, err := os.ReadFile("000018_purchase_order_draft_edit_commands.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "RAISE EXCEPTION") || strings.Contains(string(down), "DELETE FROM") {
		t.Fatal("rollback must refuse completed edits, not erase evidence")
	}
}

func TestVarianceResolutionMigrationRetainsActorScopedEvidence(t *testing.T) {
	up, err := files.ReadFile("000019_receipt_variance_resolution_results.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, guard := range []string{"PRIMARY KEY(actor_id,idempotency_key)", "REFERENCES goods_receipts(id) ON DELETE RESTRICT", "variance_id UUID NOT NULL", "expected_version BIGINT", "result_snapshot JSONB NOT NULL", "request_hash VARCHAR(64)"} {
		if !strings.Contains(string(up), guard) {
			t.Fatalf("missing variance evidence guard: %s", guard)
		}
	}
	down, err := os.ReadFile("000019_receipt_variance_resolution_results.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "RAISE EXCEPTION") || strings.Contains(string(down), "DELETE FROM") {
		t.Fatal("rollback must not erase completed variance decisions")
	}
}

func TestGoodsReceivedOutboxMigrationSupportsLeasedRetries(t *testing.T) {
	sqlBytes, err := files.ReadFile("000013_goods_received_outbox.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"goods_received_outbox", "event_id TEXT PRIMARY KEY", "next_attempt_at", "claimed_until", "attempts", "last_error", "published_at", "goods_received_outbox_pending_idx"} {
		if !strings.Contains(sql, required) {
			t.Errorf("goods received outbox migration missing %q", required)
		}
	}
}

func TestInspectionIdempotencyMigrationKeepsReplayEvidenceOnReceipt(t *testing.T) {
	encoded, err := files.ReadFile("000014_receipt_inspection_idempotency.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"ALTER TABLE goods_receipts", "inspection_idempotency_key VARCHAR(128)", "inspection_request_hash VARCHAR(64)", "goods_receipts_inspection_idempotency_valid", "status = 'INSPECTED'", "inspected_by IS NOT NULL"} {
		if !strings.Contains(string(encoded), required) {
			t.Fatalf("missing migration guard: %s", required)
		}
	}
}

func TestPurchaseOrderIdempotencyMigrationAddsActorScopedUniqueKey(t *testing.T) {
	sqlBytes, err := files.ReadFile("000012_purchase_order_idempotency.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"idempotency_key VARCHAR(128)", "idempotency_request_hash VARCHAR(64)", "purchase_orders_created_by_idempotency_key_unique", "ON purchase_orders (created_by, idempotency_key)"} {
		if !strings.Contains(sql, required) {
			t.Errorf("purchase order idempotency migration missing %q", required)
		}
	}
}

func TestSupplyPlanConversionMigrationLinksOnlyOnePOPerPlan(t *testing.T) {
	sqlBytes, err := files.ReadFile("000011_supply_plan_purchase_order_conversion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"'CONVERTED'", "converted_purchase_order_id", "supply_plan_id UUID REFERENCES supply_plans(id)", "supply_plan_revision", "CREATE UNIQUE INDEX purchase_orders_supply_plan_unique_idx"} {
		if !strings.Contains(sql, required) {
			t.Errorf("supply plan conversion migration missing %q", required)
		}
	}
}

func TestSupplyPlanApprovalMigrationAddsLifecycleAndAuditHistory(t *testing.T) {
	sqlBytes, err := files.ReadFile("000010_supply_plan_approval_workflow.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"PENDING_APPROVAL", "APPROVED", "REJECTED", "submitted_by", "approved_by", "rejected_by", "CREATE TABLE supply_plan_status_history", "UNIQUE (supply_plan_id, version)"} {
		if !strings.Contains(sql, required) {
			t.Errorf("supply plan approval migration missing %q", required)
		}
	}
}

func TestSupplyPlanRevisionMutationIsRejectedByDatabase(t *testing.T) {
	sqlBytes, err := files.ReadFile("000009_supply_plan_revision_immutability.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"BEFORE UPDATE OR DELETE ON supply_plan_revisions", "RAISE EXCEPTION 'supply plan revisions are immutable'"} {
		if !strings.Contains(sql, required) {
			t.Errorf("revision immutability migration missing %q", required)
		}
	}
}

func TestSupplyPlanMigrationStoresImmutableRevisionSnapshots(t *testing.T) {
	sqlBytes, err := files.ReadFile("000008_supply_plan_snapshots.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	for _, required := range []string{"CREATE TABLE IF NOT EXISTS supply_plans", "current_revision", "CREATE TABLE IF NOT EXISTS supply_plan_revisions", "snapshot JSONB", "PRIMARY KEY (supply_plan_id, revision)"} {
		if !strings.Contains(sql, required) {
			t.Errorf("supply plan migration missing %q", required)
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
