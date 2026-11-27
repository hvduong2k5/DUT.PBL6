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

func assertMasterDataMutationE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()
	repo := postgres.New(db)
	service := application.NewService(repo)
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	beforeEvents := count(`SELECT count(*) FROM goods_received_outbox`)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	supplier, _ := domain.NewSupplier("MUTATION-SUP", "Mutation supplier", "COMPANY", "", "", "", "")
	supplier, err := service.CreateSupplier(ctx, supplier)
	if err != nil {
		t.Fatal(err)
	}
	material, _ := domain.NewRawMaterial("MUTATION-MAT", "Mutation material", "KG", "")
	material, err = service.CreateMaterial(ctx, material)
	if err != nil {
		t.Fatal(err)
	}
	offer, _ := domain.NewSupplierMaterialOfferVersion(supplier.ID, material.ID, "", "KG", 10000, "VND", "1", 1, time.Now().UTC().Add(-48*time.Hour), nil)
	offer, err = service.CreateOffer(ctx, offer)
	if err != nil {
		t.Fatal(err)
	}
	currentSupplier := func() domain.Supplier {
		t.Helper()
		v, e := repo.GetSupplier(ctx, supplier.ID)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	currentMaterial := func() domain.RawMaterial {
		t.Helper()
		v, e := repo.GetMaterial(ctx, material.ID)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	currentOffer := func() domain.SupplierMaterialOffer {
		t.Helper()
		rows, e := repo.ListSupplierOffers(ctx, supplier.ID)
		if e != nil {
			t.Fatal(e)
		}
		for _, v := range rows {
			if v.ID == offer.ID {
				return v
			}
		}
		t.Fatal("offer missing")
		return domain.SupplierMaterialOffer{}
	}
	command := func(action, key string) domain.MasterDataMutation {
		c := domain.MasterDataMutation{Action: action, Actor: "master-mutation-actor", Key: key}
		switch action {
		case "EDIT_SUPPLIER", "SUPPLIER_STATUS":
			v := currentSupplier()
			c.ResourceID = v.ID
			c.ExpectedUpdatedAt = v.UpdatedAt
			c.Supplier = &v
		case "OFFER_STATUS":
			v := currentOffer()
			c.ResourceID = v.ID
			c.SupplierID = v.SupplierID
			c.ExpectedUpdatedAt = v.UpdatedAt
		default:
			v := currentMaterial()
			c.ResourceID = v.ID
			c.ExpectedUpdatedAt = v.UpdatedAt
			c.Material = &v
		}
		return c
	}
	run := func(c domain.MasterDataMutation) json.RawMessage {
		t.Helper()
		data, e := service.MutateMasterData(ctx, c)
		if e != nil {
			t.Fatalf("%s: %v", c.Action, e)
		}
		return data
	}
	same := func(a, b json.RawMessage) bool {
		var x, y any
		_ = json.Unmarshal(a, &x)
		_ = json.Unmarshal(b, &y)
		return reflect.DeepEqual(x, y)
	}
	deny := func(c domain.MasterDataMutation, want error) {
		t.Helper()
		if _, e := service.MutateMasterData(ctx, c); !errors.Is(e, want) {
			t.Fatalf("%s error=%v want=%v", c.Action, e, want)
		}
	}
	faultOn := func() {
		exec(`CREATE FUNCTION reject_master_mutation_result() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected master mutation failure'; END $$; CREATE TRIGGER reject_master_mutation_result BEFORE INSERT ON master_data_mutation_results FOR EACH ROW EXECUTE FUNCTION reject_master_mutation_result()`)
	}
	faultOff := func() {
		exec(`DROP TRIGGER reject_master_mutation_result ON master_data_mutation_results; DROP FUNCTION reject_master_mutation_result()`)
	}
	// Full edits preserve immutable identifiers/status and roll back with journal failure.
	editSupplier := command("EDIT_SUPPLIER", "master-mutation-supplier-0001")
	editSupplier.Supplier.Name = "Edited supplier"
	editMaterial := command("EDIT_MATERIAL", "master-mutation-material-0001")
	editMaterial.Material.Description = "Edited description"
	for _, c := range []domain.MasterDataMutation{editSupplier, editMaterial} {
		faultOn()
		if _, e := service.MutateMasterData(ctx, c); e == nil {
			t.Fatal("journal failure committed edit")
		}
		faultOff()
	}
	if currentSupplier().Name != supplier.Name || currentMaterial().Description != "" || count(`SELECT count(*) FROM master_data_mutation_results`) != 0 {
		t.Fatal("failed edit left changes/evidence")
	}
	supplierEditResult := run(editSupplier)
	run(editMaterial)
	if currentSupplier().Code != supplier.Code || currentMaterial().BaseUnit != material.BaseUnit || !currentSupplier().CreatedAt.Equal(supplier.CreatedAt) {
		t.Fatal("edit changed identity/creation data")
	}
	for _, c := range []domain.MasterDataMutation{editSupplier, editMaterial} {
		c.Key += "-stale"
		deny(c, domain.ErrConflict)
	}
	immutable := command("EDIT_MATERIAL", "master-mutation-unit-0001")
	immutable.Material.BaseUnit = "G"
	deny(immutable, domain.ErrConflict)
	immutable = command("EDIT_SUPPLIER", "master-mutation-code-0001")
	immutable.Supplier.Code = "CHANGED"
	deny(immutable, domain.ErrConflict)
	// Setting/clearing preference, missing/ineligible supplier and rollback.
	preference := command("PREFERRED_SUPPLIER", "master-mutation-prefer-0001")
	preference.PreferredSupplierID = supplier.ID
	missing := preference
	missing.Key += "-missing"
	missing.PreferredSupplierID = uuid.New()
	deny(missing, domain.ErrNotFound)
	faultOn()
	if _, e := service.MutateMasterData(ctx, preference); e == nil {
		t.Fatal("failed preference committed")
	}
	faultOff()
	if currentMaterial().PreferredSupplierID != nil {
		t.Fatal("preference leaked from failed tx")
	}
	prefResult := run(preference)
	// Suspension and implicit preference cleanup commit/rollback together.
	suspended := command("SUPPLIER_STATUS", "master-mutation-suspend-0001")
	suspended.Status = "SUSPENDED"
	beforeMaterial := currentMaterial()
	faultOn()
	if _, e := service.MutateMasterData(ctx, suspended); e == nil {
		t.Fatal("failed suspension committed")
	}
	faultOff()
	if currentSupplier().Status != domain.SupplierActive || currentMaterial().PreferredSupplierID == nil || !currentMaterial().UpdatedAt.Equal(beforeMaterial.UpdatedAt) {
		t.Fatal("suspension cleanup was not atomic")
	}
	run(suspended)
	if currentMaterial().PreferredSupplierID != nil || !currentMaterial().UpdatedAt.After(beforeMaterial.UpdatedAt) {
		t.Fatal("cleanup did not invalidate material token")
	}
	deniedPref := command("PREFERRED_SUPPLIER", "master-mutation-inactive-supplier")
	deniedPref.PreferredSupplierID = supplier.ID
	deny(deniedPref, domain.ErrInvalidState)
	// Old frozen result still available after later edits/status/implicit cleanup.
	service = application.NewService(postgres.New(db))
	if !same(supplierEditResult, run(editSupplier)) || !same(prefResult, run(preference)) {
		t.Fatal("restart did not preserve frozen mutation result")
	}
	active := command("SUPPLIER_STATUS", "master-mutation-reactivate-0001")
	active.Status = "ACTIVE"
	run(active)
	if currentMaterial().PreferredSupplierID != nil {
		t.Fatal("reactivation restored preference")
	}
	preference = command("PREFERRED_SUPPLIER", "master-mutation-prefer-0002")
	preference.PreferredSupplierID = supplier.ID
	run(preference)
	// Offer toggle keeps price version/terms and rolls back preference side effects.
	disabled := command("OFFER_STATUS", "master-mutation-offer-off-0001")
	off := false
	disabled.Active = &off
	faultOn()
	if _, e := service.MutateMasterData(ctx, disabled); e == nil {
		t.Fatal("failed offer toggle committed")
	}
	faultOff()
	if !currentOffer().Active || currentMaterial().PreferredSupplierID == nil {
		t.Fatal("failed toggle lost offer/preference")
	}
	offerResult := run(disabled)
	if currentOffer().Active || currentMaterial().PreferredSupplierID != nil || currentOffer().Version != offer.Version || currentOffer().UnitPriceMinor != offer.UnitPriceMinor {
		t.Fatal("offer toggle changed terms or missed cleanup")
	}
	enabled := command("OFFER_STATUS", "master-mutation-offer-on-0001")
	on := true
	enabled.Active = &on
	run(enabled)
	if !same(offerResult, run(disabled)) || !currentOffer().Active {
		t.Fatal("old toggle replay changed current offer")
	}
	// Material inactivity also clears preference; clearing remains valid when inactive.
	preference = command("PREFERRED_SUPPLIER", "master-mutation-prefer-0003")
	preference.PreferredSupplierID = supplier.ID
	run(preference)
	inactive := command("MATERIAL_STATUS", "master-mutation-material-off-0001")
	inactive.Status = "INACTIVE"
	faultOn()
	if _, e := service.MutateMasterData(ctx, inactive); e == nil {
		t.Fatal("failed material status committed")
	}
	faultOff()
	if currentMaterial().Status != domain.MaterialActive || currentMaterial().PreferredSupplierID == nil {
		t.Fatal("failed inactive cleanup committed")
	}
	run(inactive)
	if currentMaterial().PreferredSupplierID != nil {
		t.Fatal("inactive material retained preference")
	}
	deniedPref = command("PREFERRED_SUPPLIER", "master-mutation-inactive-material")
	deniedPref.PreferredSupplierID = supplier.ID
	deny(deniedPref, domain.ErrInvalidState)
	run(command("PREFERRED_SUPPLIER", "master-mutation-clear-inactive"))
	activeMaterial := command("MATERIAL_STATUS", "master-mutation-material-on-0001")
	activeMaterial.Status = "ACTIVE"
	run(activeMaterial)
	// Active ranges cannot overlap when re-enabling a previously disabled price.
	disabled = command("OFFER_STATUS", "master-mutation-offer-off-0002")
	disabled.Active = &off
	run(disabled)
	another := offer
	another.ID = uuid.New()
	another.UnitPriceMinor = 11000
	another, err = service.CreateOffer(ctx, another)
	if err != nil {
		t.Fatal(err)
	}
	enabled = command("OFFER_STATUS", "master-mutation-overlap-enable")
	enabled.Active = &on
	deny(enabled, domain.ErrConflict)
	wrongParent := enabled
	wrongParent.Key += "-parent"
	wrongParent.SupplierID = uuid.New()
	deny(wrongParent, domain.ErrNotFound)
	absent := enabled
	absent.Key += "-missing"
	absent.ResourceID = uuid.New()
	deny(absent, domain.ErrNotFound)
	// Competing edits with one observed timestamp cannot overwrite one another.
	a := command("EDIT_SUPPLIER", "master-mutation-race-edit-0001")
	a.Supplier.Name = "Race A"
	b := a
	copySupplier := *a.Supplier
	copySupplier.Name = "Race B"
	b.Supplier = &copySupplier
	b.Key = "master-mutation-race-edit-0002"
	type outcome struct {
		data json.RawMessage
		err  error
	}
	results := make(chan outcome, 4)
	start := make(chan struct{})
	for _, c := range []domain.MasterDataMutation{a, b} {
		go func(v domain.MasterDataMutation) {
			<-start
			data, e := service.MutateMasterData(ctx, v)
			results <- outcome{data, e}
		}(c)
	}
	close(start)
	success, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		v := <-results
		if v.err == nil {
			success++
		} else if errors.Is(v.err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(v.err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("stale competing edit overwritten")
	}
	a = command("EDIT_SUPPLIER", "master-mutation-same-key-0001")
	a.Supplier.Name = "Same key"
	start = make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() { <-start; data, e := service.MutateMasterData(ctx, a); results <- outcome{data, e} }()
	}
	close(start)
	var first json.RawMessage
	for i := 0; i < 4; i++ {
		v := <-results
		if v.err != nil {
			t.Fatal(v.err)
		}
		if first == nil {
			first = v.data
		} else if !same(first, v.data) {
			t.Fatal("same-key concurrency changed result")
		}
	}
	cross := command("MATERIAL_STATUS", a.Key)
	cross.Status = "ACTIVE"
	deny(cross, domain.ErrConflict)
	// No-op and attempted clock rollback still advance the optimistic token.
	noOp := command("SUPPLIER_STATUS", "master-mutation-noop-0001")
	noOp.Status = "ACTIVE"
	run(noOp)
	if !currentSupplier().UpdatedAt.After(noOp.ExpectedUpdatedAt) {
		t.Fatal("no-op did not advance token")
	}
	stamp := currentSupplier().UpdatedAt
	exec(`UPDATE suppliers SET updated_at='1900-01-01' WHERE id=$1`, supplier.ID)
	if !currentSupplier().UpdatedAt.After(stamp) {
		t.Fatal("clock rollback defeated token")
	}
	if count(`SELECT count(*) FROM goods_received_outbox`) != beforeEvents {
		t.Fatal("master change emitted receiving events")
	}
	t.Log("master mutations: all six actions, journal/cleanup rollback, frozen restart replay, optimistic races, immutable code/unit, overlap and monotonic timestamps PASS")
}
