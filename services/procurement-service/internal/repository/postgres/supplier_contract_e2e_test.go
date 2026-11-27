package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/repository/postgres"
)

func assertSupplierContractE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
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
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	beforeOutbox := count(`SELECT count(*) FROM goods_received_outbox`)
	supplier, _ := domain.NewSupplier("CONTRACT-DB-SUP", "Contract DB supplier", "COMPANY", "", "", "", "")
	supplier, err := service.CreateSupplier(ctx, supplier)
	if err != nil {
		t.Fatal(err)
	}
	from := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -2)
	contract, _ := domain.NewSupplierContract(supplier.ID, "AGR-DB", " raw_materials ", "NET 30", "", from, nil, "forged-client-actor")
	create := domain.SupplierContractCommand{Action: "CREATE", Actor: "contract-db-actor", Key: "contract-db-create-0001", Contract: &contract}
	run := func(c domain.SupplierContractCommand) json.RawMessage {
		t.Helper()
		data, e := service.ExecuteSupplierContractCommand(ctx, c)
		if e != nil {
			t.Fatalf("%s: %v", c.Action, e)
		}
		return data
	}
	current := func(id uuid.UUID) domain.SupplierContract {
		t.Helper()
		v, e := repo.GetSupplierContract(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	decode := func(data json.RawMessage) domain.SupplierContract {
		t.Helper()
		var v domain.SupplierContract
		if e := json.Unmarshal(data, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	same := func(a, b json.RawMessage) bool {
		var x, y any
		_ = json.Unmarshal(a, &x)
		_ = json.Unmarshal(b, &y)
		return reflect.DeepEqual(x, y)
	}
	deny := func(c domain.SupplierContractCommand, want error) {
		t.Helper()
		if _, e := service.ExecuteSupplierContractCommand(ctx, c); !errors.Is(e, want) {
			t.Fatalf("%s got %v want %v", c.Action, e, want)
		}
	}
	faultOn := func() {
		exec(`CREATE FUNCTION reject_contract_result() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected contract result failure'; END $$; CREATE TRIGGER reject_contract_result BEFORE INSERT ON supplier_contract_command_results FOR EACH ROW EXECUTE FUNCTION reject_contract_result()`)
	}
	faultOff := func() {
		exec(`DROP TRIGGER reject_contract_result ON supplier_contract_command_results; DROP FUNCTION reject_contract_result()`)
	}
	fault := func(c domain.SupplierContractCommand) {
		t.Helper()
		faultOn()
		_, e := service.ExecuteSupplierContractCommand(ctx, c)
		faultOff()
		if e == nil || !strings.Contains(e.Error(), "injected contract result failure") {
			t.Fatalf("command did not reach injected journal failure: %v", e)
		}
	}
	fault(create)
	if count(`SELECT count(*) FROM supplier_contracts WHERE supplier_id=$1`, supplier.ID) != 0 || count(`SELECT count(*) FROM supplier_contract_status_history h JOIN supplier_contracts c ON c.id=h.supplier_contract_id WHERE c.supplier_id=$1`, supplier.ID) != 0 {
		t.Fatal("create rollback left revision/history")
	}
	created := run(create)
	first := decode(created)
	if first.CreatedBy != create.Actor || first.Scope != "RAW_MATERIALS" || first.Revision != 1 {
		t.Fatal("creator/scope/revision incorrect")
	}
	service = application.NewService(postgres.New(db))
	if !same(created, run(create)) {
		t.Fatal("restart create replay changed")
	}
	changed := contract
	changed.PaymentTerms = "NET 60"
	conflict := create
	conflict.Contract = &changed
	deny(conflict, domain.ErrConflict)
	unverified := contract
	unverified.DocumentFileID = "unverified-file"
	invalid := create
	invalid.Key += "-file"
	invalid.Contract = &unverified
	deny(invalid, domain.ErrInvalidArgument)
	absent := contract
	absent.SupplierID = uuid.New()
	invalid = create
	invalid.Key += "-missing"
	invalid.Contract = &absent
	deny(invalid, domain.ErrNotFound)
	// Matching concurrent requests produce only one revision/history/result.
	type outcome struct {
		data json.RawMessage
		err  error
	}
	results := make(chan outcome, 4)
	start := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			<-start
			data, e := service.ExecuteSupplierContractCommand(ctx, create)
			results <- outcome{data, e}
		}()
	}
	close(start)
	for i := 0; i < 4; i++ {
		v := <-results
		if v.err != nil || !same(created, v.data) {
			t.Fatalf("concurrent replay: %v", v.err)
		}
	}
	if count(`SELECT count(*) FROM supplier_contracts WHERE supplier_id=$1`, supplier.ID) != 1 || count(`SELECT count(*) FROM supplier_contract_status_history WHERE supplier_contract_id=$1`, first.ID) != 1 {
		t.Fatal("concurrent retry duplicated revision/history")
	}
	// New keys intentionally create new immutable revisions, with no duplicate numbers.
	start = make(chan struct{})
	for _, key := range []string{"contract-db-revision-0002", "contract-db-revision-0003"} {
		c := create
		c.Key = key
		go func() {
			<-start
			data, e := service.ExecuteSupplierContractCommand(ctx, c)
			results <- outcome{data, e}
		}()
	}
	close(start)
	revisions := map[int]domain.SupplierContract{1: first}
	for i := 0; i < 2; i++ {
		v := <-results
		if v.err != nil {
			t.Fatal(v.err)
		}
		c := decode(v.data)
		revisions[c.Revision] = c
	}
	if len(revisions) != 3 {
		t.Fatal("parallel revision allocation collided")
	}
	page, e := service.ListSupplierContracts(ctx, supplier.ID, 1, 1)
	if e != nil || len(page) != 1 || page[0].Revision != 2 {
		t.Fatalf("revision pagination: %v %v", page, e)
	}
	activate := domain.SupplierContractCommand{Action: "ACTIVATE", Actor: "contract-db-actor", Key: "contract-db-activate-0001", ContractID: first.ID, ExpectedUpdatedAt: first.UpdatedAt}
	fault(activate)
	if current(first.ID).Status != domain.SupplierContractDraft || count(`SELECT count(*) FROM supplier_contract_status_history WHERE supplier_contract_id=$1`, first.ID) != 1 {
		t.Fatal("failed activation left status/history")
	}
	activeSnapshot := run(activate)
	active := decode(activeSnapshot)
	if !active.UpdatedAt.After(first.UpdatedAt) {
		t.Fatal("activation did not advance timestamp")
	}
	cross := activate
	cross.Key = create.Key
	deny(cross, domain.ErrConflict)
	stale := activate
	stale.Key += "-stale"
	deny(stale, domain.ErrConflict)
	// Overlap rejection applies across numbers and revisions for the same scope.
	second := revisions[2]
	overlap := activate
	overlap.ContractID = second.ID
	overlap.ExpectedUpdatedAt = second.UpdatedAt
	overlap.Key = "contract-db-overlap-0001"
	deny(overlap, domain.ErrConflict)
	// Two terminators observing the same token: exactly one commits its reason/history.
	terminate := domain.SupplierContractCommand{Action: "TERMINATE", Actor: "contract-db-actor", Key: "contract-db-terminate-0001", ContractID: first.ID, ExpectedUpdatedAt: active.UpdatedAt, Reason: " Supplier partnership ended "}
	fault(terminate)
	if current(first.ID).Status != domain.SupplierContractActive || count(`SELECT count(*) FROM supplier_contract_status_history WHERE supplier_contract_id=$1`, first.ID) != 2 {
		t.Fatal("failed termination left status/history")
	}
	start = make(chan struct{})
	for _, key := range []string{terminate.Key, terminate.Key + "-competing"} {
		c := terminate
		c.Key = key
		go func() {
			<-start
			data, e := service.ExecuteSupplierContractCommand(ctx, c)
			results <- outcome{data, e}
		}()
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
		t.Fatal("competing termination overwrote reason")
	}
	if count(`SELECT count(*) FROM supplier_contract_status_history WHERE supplier_contract_id=$1 AND to_status='TERMINATED' AND actor_id=$2 AND note='Supplier partnership ended'`, first.ID, terminate.Actor) != 1 {
		t.Fatal("termination reason/actor not audited once")
	}
	if !same(created, run(create)) || !same(activeSnapshot, run(activate)) || current(first.ID).Status != domain.SupplierContractTerminated {
		t.Fatal("old frozen replay changed live lifecycle")
	}
	// A terminated contract cannot reactivate; terms cannot be changed or removed in SQL.
	noReactivate := activate
	noReactivate.Key = "contract-db-reactivate-denied"
	noReactivate.ExpectedUpdatedAt = current(first.ID).UpdatedAt
	deny(noReactivate, domain.ErrInvalidState)
	for _, sql := range []string{`UPDATE supplier_contracts SET payment_terms='Changed' WHERE id=$1`, `DELETE FROM supplier_contracts WHERE id=$1`} {
		if _, e := db.Exec(ctx, sql, first.ID); e == nil {
			t.Fatal("immutable contract evidence changed")
		}
	}
	if _, e := db.Exec(ctx, `TRUNCATE supplier_contracts CASCADE`); e == nil {
		t.Fatal("contract evidence truncated")
	}
	// Competing activation for still-draft overlapping agreements is serialized.
	a := overlap
	a.Key = "contract-db-activate-race-0001"
	third := revisions[3]
	b := a
	b.Key = "contract-db-activate-race-0002"
	b.ContractID = third.ID
	b.ExpectedUpdatedAt = third.UpdatedAt
	start = make(chan struct{})
	for _, c := range []domain.SupplierContractCommand{a, b} {
		go func(v domain.SupplierContractCommand) {
			<-start
			data, e := service.ExecuteSupplierContractCommand(ctx, v)
			results <- outcome{data, e}
		}(c)
	}
	close(start)
	success, conflicts = 0, 0
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
		t.Fatal("overlapping active agreements committed")
	}
	// Expired drafts and inactive suppliers cannot activate/create fresh agreements.
	expired := contract
	end := from.Add(24 * time.Hour)
	expired.EffectiveTo = &end
	expired.ContractNumber = "EXPIRED"
	expiredCmd := create
	expiredCmd.Key = "contract-db-expired-create"
	expiredCmd.Contract = &expired
	expiredContract := decode(run(expiredCmd))
	badActivate := activate
	badActivate.Key = "contract-db-expired-activate"
	badActivate.ContractID = expiredContract.ID
	badActivate.ExpectedUpdatedAt = expiredContract.UpdatedAt
	deny(badActivate, domain.ErrInvalidState)
	if _, e := service.SetSupplierStatus(ctx, supplier.ID, domain.SupplierSuspended); e != nil {
		t.Fatal(e)
	}
	inactive := create
	inactive.Key = "contract-db-inactive-create"
	deny(inactive, domain.ErrInvalidState)
	if !same(created, run(create)) {
		t.Fatal("inactive supplier prevented committed replay")
	}
	if count(`SELECT count(*) FROM goods_received_outbox`) != beforeOutbox {
		t.Fatal("contract command emitted receiving events")
	}
	t.Log("supplier contracts: atomic create/status/history/result, frozen replay, revision races, overlap, termination audit, immutable terms and expiry/parent guards PASS")
}
