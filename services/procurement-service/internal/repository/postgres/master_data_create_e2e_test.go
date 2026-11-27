package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/repository/postgres"
)

// Invoked only inside the disposable receiving-test schema. No receiving events
// are created here and no live Inventory/master records are changed.
func assertMasterDataCreateE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()
	service := application.NewService(postgres.New(db))
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	create := func(cmd domain.MasterDataCreateCommand) json.RawMessage {
		t.Helper()
		result, err := service.CreateMasterDataWithIdempotency(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	same := func(a, b json.RawMessage) bool {
		var x, y any
		if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
			return false
		}
		return reflect.DeepEqual(x, y)
	}
	supplier, _ := domain.NewSupplierWithLegalIdentifier("MASTER-DB-SUP", "Master DB supplier", "COMPANY", "", "", "", "", "MASTER-DB-TAX")
	material, _ := domain.NewRawMaterial("MASTER-DB-MAT", "Master DB material", "KG", "")
	supplierCmd := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-db-supplier-0001", Kind: "SUPPLIER", Supplier: &supplier}
	materialCmd := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-db-material-0001", Kind: "MATERIAL", Material: &material}
	// Result insertion failure must undo the business INSERT and leave the key free.
	faultOn := func() {
		t.Helper()
		_, err := db.Exec(ctx, `CREATE FUNCTION reject_master_create_result() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected master result failure'; END $$; CREATE TRIGGER reject_master_create_result BEFORE INSERT ON master_data_create_results FOR EACH ROW EXECUTE FUNCTION reject_master_create_result()`)
		if err != nil {
			t.Fatal(err)
		}
	}
	faultOff := func() {
		t.Helper()
		if _, err := db.Exec(ctx, `DROP TRIGGER reject_master_create_result ON master_data_create_results; DROP FUNCTION reject_master_create_result()`); err != nil {
			t.Fatal(err)
		}
	}
	faultOn()
	for _, cmd := range []domain.MasterDataCreateCommand{supplierCmd, materialCmd} {
		if _, err := service.CreateMasterDataWithIdempotency(ctx, cmd); err == nil {
			t.Fatal("journal failure accepted")
		}
	}
	if count(`SELECT count(*) FROM suppliers WHERE code=$1`, supplier.Code) != 0 || count(`SELECT count(*) FROM raw_materials WHERE code=$1`, material.Code) != 0 || count(`SELECT count(*) FROM master_data_create_results WHERE actor_id='master-actor'`) != 0 {
		t.Fatal("failed create left business/journal evidence")
	}
	faultOff()
	supplierResult := create(supplierCmd)
	materialResult := create(materialCmd)
	if err := json.Unmarshal(supplierResult, &supplier); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(materialResult, &material); err != nil {
		t.Fatal(err)
	}
	// A fresh service/repository replays normalized input despite fresh generated IDs.
	service = application.NewService(postgres.New(db))
	supplierCmd.Supplier.Code = " master-db-sup "
	supplierCmd.Supplier.LegalIdentifier = " master-db-tax "
	if !same(supplierResult, create(supplierCmd)) {
		t.Fatal("restart/normalization replay changed snapshot")
	}
	if _, err := service.CreateMasterDataWithIdempotency(ctx, domain.MasterDataCreateCommand{Actor: supplierCmd.Actor, Key: supplierCmd.Key, Kind: "MATERIAL", Material: &material}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("cross-kind collision: %v", err)
	}
	changed := *supplierCmd.Supplier
	changed.Name = "Changed"
	cmd := supplierCmd
	cmd.Supplier = &changed
	if _, err := service.CreateMasterDataWithIdempotency(ctx, cmd); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("changed payload: %v", err)
	}
	duplicate := supplierCmd
	duplicate.Key += "-other"
	if _, err := service.CreateMasterDataWithIdempotency(ctx, duplicate); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("unique supplier code: %v", err)
	}
	// Simultaneous identical requests serialize before row creation and return one ID.
	concurrentMaterial, _ := domain.NewRawMaterial("MASTER-DB-RACE", "Race material", "KG", "")
	concurrentCmd := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-db-race-key-0001", Kind: "MATERIAL", Material: &concurrentMaterial}
	type outcome struct {
		data json.RawMessage
		err  error
	}
	results := make(chan outcome, 6)
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		go func() {
			<-start
			data, err := service.CreateMasterDataWithIdempotency(ctx, concurrentCmd)
			results <- outcome{data, err}
		}()
	}
	close(start)
	var first json.RawMessage
	for i := 0; i < 6; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if first == nil {
			first = r.data
		} else if !same(first, r.data) {
			t.Fatal("concurrent retry changed ID")
		}
	}
	if count(`SELECT count(*) FROM raw_materials WHERE code=$1`, concurrentMaterial.Code) != 1 {
		t.Fatal("duplicate material created")
	}
	// Different create kinds with the same actor/key cannot both commit.
	raceSupplier, _ := domain.NewSupplier("MASTER-CROSS-SUP", "Cross supplier", "COMPANY", "", "", "", "")
	raceMaterial, _ := domain.NewRawMaterial("MASTER-CROSS-MAT", "Cross material", "KG", "")
	cross := []domain.MasterDataCreateCommand{{Actor: "master-cross", Key: "master-cross-key-0001", Kind: "SUPPLIER", Supplier: &raceSupplier}, {Actor: "master-cross", Key: "master-cross-key-0001", Kind: "MATERIAL", Material: &raceMaterial}}
	start = make(chan struct{})
	for _, v := range cross {
		go func(c domain.MasterDataCreateCommand) {
			<-start
			data, err := service.CreateMasterDataWithIdempotency(ctx, c)
			results <- outcome{data, err}
		}(v)
	}
	close(start)
	success, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			success++
		} else if errors.Is(r.err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(r.err)
		}
	}
	if success != 1 || conflicts != 1 || count(`SELECT count(*) FROM suppliers WHERE code='MASTER-CROSS-SUP'`)+count(`SELECT count(*) FROM raw_materials WHERE code='MASTER-CROSS-MAT'`) != 1 {
		t.Fatal("cross-kind key was not atomic")
	}
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	offer, _ := domain.NewSupplierMaterialOfferVersion(supplier.ID, material.ID, "", "KG", 10000, "VND", "1", 1, from, nil)
	offerCmd := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-db-offer-0001", Kind: "OFFER", Offer: &offer}
	originalOffer := create(offerCmd)
	next := offer
	next.ID = uuid.New()
	next.EffectiveFrom = from.AddDate(0, 1, 0)
	next.UnitPriceMinor = 11000
	nextCmd := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-db-offer-0002", Kind: "OFFER", Offer: &next}
	faultOn()
	if _, err := service.CreateMasterDataWithIdempotency(ctx, nextCmd); err == nil {
		t.Fatal("offer journal failure accepted")
	}
	faultOff()
	if count(`SELECT count(*) FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2 AND effective_to IS NULL`, supplier.ID, material.ID) != 1 || count(`SELECT count(*) FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2`, supplier.ID, material.ID) != 1 {
		t.Fatal("offer fault did not restore previous window")
	}
	nextResult := create(nextCmd)
	var v2 domain.SupplierMaterialOffer
	_ = json.Unmarshal(nextResult, &v2)
	if v2.Version != 2 || count(`SELECT count(*) FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2 AND version=1 AND effective_to=$3`, supplier.ID, material.ID, next.EffectiveFrom) != 1 {
		t.Fatal("version/window did not commit together")
	}
	// Frozen replay must precede current parent/window checks.
	if _, err := db.Exec(ctx, `UPDATE suppliers SET status='SUSPENDED' WHERE id=$1`, supplier.ID); err != nil {
		t.Fatal(err)
	}
	if !same(supplierResult, create(supplierCmd)) || !same(originalOffer, create(offerCmd)) {
		t.Fatal("mutable resource changed frozen create snapshot")
	}
	rejected := nextCmd
	rejected.Key = "master-inactive-offer-0001"
	if _, err := service.CreateMasterDataWithIdempotency(ctx, rejected); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("inactive supplier: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE suppliers SET status='ACTIVE' WHERE id=$1`, supplier.ID); err != nil {
		t.Fatal(err)
	}
	rejected = nextCmd
	rejected.Key = "master-overlap-offer-0001"
	if _, err := service.CreateMasterDataWithIdempotency(ctx, rejected); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("overlap: %v", err)
	}
	// Distinct keys racing for the same effective window produce only one version.
	next.EffectiveFrom = next.EffectiveFrom.AddDate(0, 1, 0)
	start = make(chan struct{})
	for _, key := range []string{"master-price-race-0001", "master-price-race-0002"} {
		c := nextCmd
		c.Key = key
		c.Offer = &next
		go func() {
			<-start
			data, err := service.CreateMasterDataWithIdempotency(ctx, c)
			results <- outcome{data, err}
		}()
	}
	close(start)
	success, conflicts = 0, 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			success++
		} else if errors.Is(r.err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(r.err)
		}
	}
	if success != 1 || conflicts != 1 || count(`SELECT count(*) FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2`, supplier.ID, material.ID) != 3 {
		t.Fatal("concurrent price creation overlapped or skipped version")
	}
	// Invalid direct application input must never reach SQL or consume a key.
	invalid := domain.MasterDataCreateCommand{Actor: "master-actor", Key: "master-invalid-key-0001", Kind: "OFFER", Offer: &domain.SupplierMaterialOffer{}}
	if _, err := service.CreateMasterDataWithIdempotency(ctx, invalid); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	if count(`SELECT count(*) FROM master_data_create_results WHERE idempotency_key=$1`, invalid.Key) != 0 {
		t.Fatal("invalid input consumed key")
	}
	t.Log("master-data create: transaction rollback, frozen restart replay, uniqueness, cross-kind/concurrent keys, version/window races PASS")
}
