package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsAreOrderedAndIncludePurchaseOrders(t *testing.T) {
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
	if len(names) != 2 || names[0] != "000001_supplier_material_master.up.sql" || names[1] != "000002_purchase_orders.up.sql" {
		t.Fatalf("unexpected embedded migration sequence: %v", names)
	}
	for _, name := range names {
		if _, err := files.ReadFile(name); err != nil {
			t.Errorf("migration %s is not embedded: %v", name, err)
		}
	}
}
