package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/migrations"
)

type harness struct {
	db    *pgxpool.Pool
	store *Store
	ctx   context.Context
}

func database(t *testing.T, upgrade bool) *harness {
	t.Helper()
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("NOT_RUN: ORDER_CORE_TEST_DATABASE (real PostgreSQL16) is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "core_pr2_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 12
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, e := admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
		if e != nil {
			t.Error(e)
		}
	})
	if upgrade {
		for _, ddl := range []string{migrations.Schema, migrations.Observability, migrations.ProviderCursor} {
			if _, err = db.Exec(ctx, ddl); err != nil {
				t.Fatal(err)
			}
		}
		_, err = db.Exec(ctx, `CREATE TABLE order_migrations(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT clock_timestamp());INSERT INTO order_migrations(version) VALUES(1),(2),(3)`)
		if err != nil {
			t.Fatal(err)
		}
		// Deliberately retain a legacy ID mapping unlike the planned target ID.
		op, planned, id, principal := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
		if _, err = db.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash) VALUES($1,$2,$3,$4,'SUCCEEDED',$5,$6)`, op, "customer:"+principal, principal, planned, []byte("legacy-cipher"), strings.Repeat("a", 64)); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,method,status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,provider_reference)
 VALUES($1,$2,$3,$4,$5,'VIETQR','PENDING_PAYMENT','RESERVED',1,0,1,$6,'legacy-reservation',clock_timestamp()+interval '1 day',$3)`, id, op, "legacy-"+id, "customer:"+principal, principal, []byte("legacy-snapshot")); err != nil {
			t.Fatal(err)
		}
	}
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal("migration replay", err)
	}
	cipher, err := NewCipher("test-key", map[string][]byte{"test-key": bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db, cipher, bytes.Repeat([]byte{8}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return &harness{db, store, ctx}
}

func (h *harness) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := h.db.QueryRow(h.ctx, query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func snapshotInput(t *testing.T, index int) core.SnapshotInput {
	t.Helper()
	p := filepath.Join("..", "..", "..", "..", "docs", "03_api_specs", "order-snapshot.v1.examples.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct {
			Snapshot core.SnapshotInput `json:"snapshot"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f.Cases[index].Snapshot
}

func acceptanceInput(t *testing.T, index int) AcceptanceInput {
	d := snapshotInput(t, index)
	return AcceptanceInput{Owner: d.Owner, Method: d.Payment.Method, Key: uuid.NewString(), RequestHash: strings.Repeat("a", 64),
		QuoteID: uuid.NewString(), CartID: uuid.NewString(), CartRevision: 1, SelectionDigest: d.Provenance.SelectionDigest, QuoteHash: d.Provenance.QuoteHash,
		QuoteExpiresAt: time.Now().Add(time.Minute), AcceptedJSON: []byte(`{"schema_version":1,"terms":"synthetic accepted quote"}`)}
}

func prepared(t *testing.T, h *harness, index int) (Acceptance, Placement) {
	t.Helper()
	guestOnline := index == 2
	if guestOnline {
		index = 0
	}
	in := acceptanceInput(t, index)
	if guestOnline {
		in.Owner = snapshotInput(t, 1).Owner
	}
	a, err := h.store.Accept(h.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	d := snapshotInput(t, index)
	if guestOnline {
		d.Owner = in.Owner
		d.Address.Source = snapshotInput(t, 1).Address.Source
	}
	var now time.Time
	if err = h.db.QueryRow(h.ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	d.OrderID, d.OperationID, d.SnapshotID = a.PlannedOrderID, a.OperationID, uuid.NewString()
	d.Provenance.QuoteID, d.Provenance.CartID, d.Provenance.CartRevision = in.QuoteID, in.CartID, in.CartRevision
	d.Provenance.AcceptedAt, d.Provenance.CanonicalizedAt, d.PlacedAt = a.AcceptedAt, now, now
	d.Catalog.ValidUntil = now.Add(20 * time.Minute)
	d.Shipping.ValidUntil = now.Add(20 * time.Minute)
	if d.Payment.Method == "VIETQR" {
		expires := now.Add(15 * time.Minute)
		d.Payment.ExpiresAt = &expires
		connection := uuid.NewString()
		if _, err = h.db.Exec(h.ctx, `INSERT INTO payment_provider_connections(id,provider,merchant_identity,environment,secret_ref) VALUES($1,'SEPAY',$2,'TEST','test-secret-ref')`, connection, uuid.NewString()); err != nil {
			t.Fatal(err)
		}
		bankID := uuid.NewString()
		profile, _ := json.Marshal(map[string]string{"bank_account_uuid": bankID, "bank_code": "VCB", "bank_name": "Vietcombank", "account_number": "0000000000", "account_holder": "SYNTHETIC ONLY", "mode": "BANK_ACCOUNT", "currency": "VND"})
		cipher, _ := h.store.cipher.Seal("receiver:"+*d.Payment.ReceiverProfileID, profile)
		if _, err = h.db.Exec(h.ctx, `INSERT INTO payment_receiver_profiles(id,connection_id,bank_account_uuid,config_version,config_cipher,accepting_new_payments) VALUES($1,$2,$3,1,$4,true)`, *d.Payment.ReceiverProfileID, connection, bankID, cipher); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := json.Marshal(d)
	encrypted, err := h.store.cipher.Seal("canonical:"+a.OperationID, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.Exec(h.ctx, `UPDATE checkout_operations SET status='PROCESSING',stage='PLACE',canonical_cipher=$2,canonical_hash=$3 WHERE id=$1`, a.OperationID, encrypted, hash(data)); err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.Exec(h.ctx, `UPDATE workflows SET stage='PLACE' WHERE aggregate_id=$1`, a.OperationID); err != nil {
		t.Fatal(err)
	}
	owner := uuid.NewString()
	job := uuid.NewString()
	if _, err = h.db.Exec(h.ctx, `UPDATE jobs SET status='DONE' WHERE workflow_id IN(SELECT id FROM workflows WHERE aggregate_id=$1)`, a.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.Exec(h.ctx, `INSERT INTO jobs(id,kind,key,stage,status,cohort,workflow_id,input_version,input_cipher,input_hash,lane,lease_owner,lease_epoch,lease_until)
 SELECT $2,'ORDER_PLACE',$3,'PLACE','LEASED','order-core-1',id,1,$4,$5,'INTERACTIVE',$6,1,clock_timestamp()+interval '10 seconds'
 FROM workflows WHERE aggregate_id=$1`, a.OperationID, job, "checkout:"+a.OperationID+":place", encrypted, hash(data), owner); err != nil {
		t.Fatal(err)
	}
	proof := core.ResourceProof{OrderID: d.OrderID, ReservationID: uuid.NewString(), ItemsDigest: d.Provenance.ItemsDigest, WarehouseID: d.WarehouseID,
		State: "RESERVED", ResourceVersion: 1, DecisionID: uuid.NewString(), DecisionAt: now, ExpiresAt: now.Add(16 * time.Minute)}
	if index == 1 {
		proof.State = "COMMITTED"
	}
	if _, err = h.db.Exec(h.ctx, "UPDATE checkout_operations SET reservation_id=$2,reservation_expires_at=$3 WHERE id=$1", a.OperationID, proof.ReservationID, proof.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	return a, Placement{a.OperationID, LeaseFence{job, owner, 1}, proof}
}

func TestMigrationUpgradePreservesLegacy(t *testing.T) {
	h := database(t, true)
	if h.count(t, "SELECT count(*) FROM orders WHERE cohort='sandbox-3.1'") != 1 || h.count(t, "SELECT count(*) FROM order_migrations") != 6 {
		t.Fatal("legacy migration compatibility")
	}
	if _, err := h.db.Exec(h.ctx, "UPDATE orders SET version=version+1 WHERE cohort='sandbox-3.1'"); err != nil {
		t.Fatal("legacy historical association was constrained", err)
	}
}

func TestPlacementClockRegressionPreservesOperation(t *testing.T) {
	h := database(t, false)
	a, in := prepared(t, h, 0)
	var encrypted []byte
	if e := h.db.QueryRow(h.ctx, "SELECT canonical_cipher FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&encrypted); e != nil {
		t.Fatal(e)
	}
	raw, e := h.store.cipher.Open("canonical:"+a.OperationID, encrypted)
	if e != nil {
		t.Fatal(e)
	}
	var terms core.SnapshotInput
	if e = json.Unmarshal(raw, &terms); e != nil {
		t.Fatal(e)
	}
	terms.Provenance.CanonicalizedAt = terms.Provenance.CanonicalizedAt.Add(time.Hour)
	raw, _ = json.Marshal(terms)
	encrypted, e = h.store.cipher.Seal("canonical:"+a.OperationID, raw)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.db.Exec(h.ctx, "UPDATE checkout_operations SET canonical_cipher=$2,canonical_hash=$3 WHERE id=$1", a.OperationID, encrypted, hash(raw)); e != nil {
		t.Fatal(e)
	}
	if _, e = h.store.Place(h.ctx, in); !errors.Is(e, ErrClock) {
		t.Fatal("clock regression must be retryable", e)
	}
	if h.count(t, "SELECT count(*) FROM orders") != 0 || h.count(t, "SELECT count(*) FROM jobs WHERE status='LEASED'") != 1 {
		t.Fatal("clock regression changed financial or workflow identity")
	}
}

func TestConcurrentAcceptanceIdentityAndQuoteConsumption(t *testing.T) {
	h := database(t, false)
	in := acceptanceInput(t, 0)
	var wg sync.WaitGroup
	ids := make(chan string, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, e := h.store.Accept(h.ctx, in)
			if e != nil {
				errs <- e
			} else {
				ids <- a.OperationID
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatal("different replay operation")
		}
	}
	for _, table := range []string{"checkout_operations", "checkout_quote_consumptions", "checkout_submission_guards", "workflows", "jobs", "audit_records"} {
		if h.count(t, "SELECT count(*) FROM "+table) != 1 {
			t.Fatal("duplicate " + table)
		}
	}
	changed := in
	changed.RequestHash = strings.Repeat("b", 64)
	if _, err := h.store.Accept(h.ctx, changed); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	changed = in
	changed.Key = uuid.NewString()
	if _, err := h.store.Accept(h.ctx, changed); !errors.Is(err, ErrQuoteUsed) {
		t.Fatal("quote used with another key", err)
	}
	changed.QuoteID = uuid.NewString()
	if _, err := h.store.Accept(h.ctx, changed); !errors.Is(err, ErrSubmitted) {
		t.Fatal("same submission fingerprint", err)
	}
	data, err := h.store.AcceptedInput(h.ctx, first, ownerScope(in.Owner))
	if err != nil || !bytes.Equal(data, in.AcceptedJSON) {
		t.Fatal("durable accepted input", err)
	}
	if _, err = h.store.AcceptedInput(h.ctx, first, "guest:other"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("cross-owner accepted input")
	}
	if _, err = h.db.Exec(h.ctx, `UPDATE checkout_operations SET input_cipher=$2 WHERE id=$1`, first, []byte("tamper")); err == nil {
		t.Fatal("mutable accepted input")
	}
}

func TestTwoKeysRaceSameQuote(t *testing.T) {
	h := database(t, false)
	in := acceptanceInput(t, 0)
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		x := in
		x.Key = uuid.NewString()
		go func() { <-start; _, err := h.store.Accept(h.ctx, x); results <- err }()
	}
	close(start)
	success, conflict := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, ErrQuoteUsed) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 || h.count(t, "SELECT count(*) FROM checkout_operations") != 1 {
		t.Fatal("quote uniqueness lost")
	}
}

type ackFault struct {
	db          *pgxpool.Pool
	afterCommit bool
	fired       atomic.Bool
	commits     atomic.Int32
	afterHook   func()
}

func (f *ackFault) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, err := f.db.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	return &ackTx{Tx: tx, f: f}, nil
}

type ackTx struct {
	pgx.Tx
	f *ackFault
}

func (t *ackTx) Commit(ctx context.Context) error {
	t.f.commits.Add(1)
	if t.f.fired.CompareAndSwap(false, true) {
		if t.f.afterCommit {
			if err := t.Tx.Commit(ctx); err != nil {
				return err
			}
		} else {
			_ = t.Tx.Rollback(ctx)
		}
		if t.f.afterHook != nil {
			t.f.afterHook()
		}
		return io.ErrUnexpectedEOF
	}
	return t.Tx.Commit(ctx)
}

func TestCommitACKRecoveryAndUnresolvedLookup(t *testing.T) {
	for _, committed := range []bool{true, false} {
		t.Run(map[bool]string{true: "committed", false: "not-committed"}[committed], func(t *testing.T) {
			h := database(t, false)
			fault := &ackFault{db: h.db, afterCommit: committed}
			h.store.runner.DB = fault
			in := acceptanceInput(t, 0)
			a, err := h.store.Accept(h.ctx, in)
			if fault.commits.Load() != 1 {
				t.Fatal("ambiguous commit closure was retried")
			}
			if committed {
				if err != nil || a.OperationID == "" || h.count(t, "SELECT count(*) FROM checkout_operations") != 1 {
					t.Fatal("committed lookup", err)
				}
			} else {
				if !errors.Is(err, ErrCommitUnknown) || h.count(t, "SELECT count(*) FROM checkout_operations") != 0 {
					t.Fatal("false rollback certainty", err)
				}
			}
			h.store.runner.DB = h.db
			replay, err := h.store.Accept(h.ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if committed && replay.OperationID != a.OperationID {
				t.Fatal("new identity after committed ACK loss")
			}
			if h.count(t, "SELECT count(*) FROM checkout_operations") != 1 {
				t.Fatal("duplicate after replay")
			}
		})
	}
}

func TestPlacementAtomicityHydrationAndIdempotence(t *testing.T) {
	for index := 0; index < 2; index++ {
		t.Run([]string{"VIETQR", "COD"}[index], func(t *testing.T) {
			h := database(t, false)
			a, p := prepared(t, h, index)
			out, err := h.store.Place(h.ctx, p)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := h.store.LoadOrder(h.ctx, out.ID(), ownerScope(out.Owner()))
			if err != nil || loaded.Snapshot().Hash() != out.Snapshot().Hash() {
				t.Fatal("hydrate", err)
			}
			again, err := h.store.Place(h.ctx, p)
			if err != nil || again.ID() != out.ID() || again.Snapshot().Hash() != out.Snapshot().Hash() {
				t.Fatal("place replay", err)
			}
			for _, table := range []string{"orders", "order_history", "order_milestones", "event_streams", "outbox_events"} {
				if h.count(t, "SELECT count(*) FROM "+table) != 1 {
					t.Fatal("non-atomic/duplicate " + table)
				}
			}
			intents := 1
			if index == 1 {
				intents = 0
			}
			if h.count(t, "SELECT count(*) FROM payment_intents") != intents {
				t.Fatal("intent atomicity")
			}
			if h.count(t, "SELECT count(*) FROM jobs WHERE kind='CART_CLEANUP' AND status='READY'") != 1 {
				t.Fatal("cleanup obligation absent")
			}
			if index == 1 && h.count(t, "SELECT count(*) FROM jobs WHERE kind='FULFILLMENT_READY' AND status='READY'") != 1 {
				t.Fatal("COD ready intent absent")
			}
			var status string
			if err = h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil || status != "SUCCEEDED" {
				t.Fatal(err)
			}
			if _, err = h.store.LoadOrder(h.ctx, out.ID(), "customer:other"); !errors.Is(err, pgx.ErrNoRows) {
				t.Fatal("order scope")
			}
			if _, err = h.db.Exec(h.ctx, `UPDATE outbox_events SET payload_bytes=$1`, []byte(`{}`)); err == nil {
				t.Fatal("outbox payload mutable")
			}
			if _, err = h.db.Exec(h.ctx, `UPDATE orders SET snapshot_hash=$1 WHERE id=$2`, strings.Repeat("b", 64), out.ID()); err == nil {
				t.Fatal("snapshot identity mutable")
			}
			if _, err = h.db.Exec(h.ctx, `UPDATE order_history SET reason='tamper'`); err == nil {
				t.Fatal("history mutable")
			}
			var payload []byte
			if err = h.db.QueryRow(h.ctx, "SELECT payload_bytes FROM outbox_events").Scan(&payload); err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(payload, []byte("phone_e164")) || bytes.Contains(payload, []byte("recipient_name")) {
				t.Fatal("PII in event")
			}
			if directory := os.Getenv("ORDER_CORE_EVIDENCE_DIR"); directory != "" {
				if err = os.WriteFile(filepath.Join(directory, "placed-"+[]string{"VIETQR", "COD"}[index]+".json"), payload, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPlacementRollbackAndACKRecovery(t *testing.T) {
	for _, point := range []string{"payment_intents", "outbox_events"} {
		t.Run(point, func(t *testing.T) {
			h := database(t, false)
			a, p := prepared(t, h, 0)
			_, err := h.db.Exec(h.ctx, `CREATE FUNCTION reject_placement() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION USING ERRCODE='23514',MESSAGE='synthetic fault';END$$;CREATE TRIGGER reject_placement BEFORE INSERT ON `+point+` FOR EACH ROW EXECUTE FUNCTION reject_placement()`)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = h.store.Place(h.ctx, p); err == nil {
				t.Fatal("fault did not abort")
			}
			for _, table := range []string{"orders", "order_items", "payment_intents", "order_history", "order_milestones", "outbox_events", "event_streams"} {
				if h.count(t, "SELECT count(*) FROM "+table) != 0 {
					t.Fatal("partial commit " + table)
				}
			}
			if h.count(t, "SELECT count(*) FROM jobs WHERE kind='CART_CLEANUP'") != 0 {
				t.Fatal("orphan cleanup")
			}
			var status string
			_ = h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status)
			if status != "PROCESSING" {
				t.Fatal("operation falsely succeeded")
			}
			if _, err = h.db.Exec(h.ctx, "DROP TRIGGER reject_placement ON "+point); err != nil {
				t.Fatal(err)
			}
			fault := &ackFault{db: h.db, afterCommit: true}
			h.store.runner.DB = fault
			out, err := h.store.Place(h.ctx, p)
			if err != nil || out.ID() != a.PlannedOrderID || fault.commits.Load() != 1 {
				t.Fatal("placement COMMIT ACK recovery", err)
			}
		})
	}
}

func TestPlacementLeaseAndProjectionIntegrity(t *testing.T) {
	h := database(t, false)
	_, p := prepared(t, h, 0)
	bad := p
	bad.Fence.Epoch++
	if _, err := h.store.Place(h.ctx, bad); !errors.Is(err, ErrLease) {
		t.Fatal("stale epoch", err)
	}
	out, err := h.store.Place(h.ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.Exec(h.ctx, `UPDATE orders SET warehouse_id='OTHER' WHERE id=$1`, out.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err = h.store.LoadOrder(h.ctx, out.ID(), ownerScope(out.Owner())); !errors.Is(err, ErrIntegrity) {
		t.Fatal("projection integrity", err)
	}
}

func TestSQLRetryWholeTransaction(t *testing.T) {
	h := database(t, false)
	if _, err := h.db.Exec(h.ctx, "CREATE TABLE retry_probe(id integer PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"40001", "40P01"} {
		attempts := 0
		err := h.store.runner.Do(h.ctx, func(tx pgx.Tx) error {
			attempts++
			if _, e := tx.Exec(h.ctx, "INSERT INTO retry_probe VALUES(1)"); e != nil {
				return e
			}
			if attempts == 1 {
				_, e := tx.Exec(h.ctx, `DO $$BEGIN RAISE EXCEPTION USING ERRCODE='`+code+`',MESSAGE='synthetic retry';END$$`)
				return e
			}
			return nil
		})
		if err != nil || attempts != 2 || h.count(t, "SELECT count(*) FROM retry_probe") != 1 {
			t.Fatal("whole transaction retry", code, attempts, err)
		}
		if _, err = h.db.Exec(h.ctx, "DELETE FROM retry_probe"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRealPostgresDeadlockRecovery(t *testing.T) {
	h := database(t, false)
	if _, err := h.db.Exec(h.ctx, "CREATE TABLE deadlock_probe(id integer PRIMARY KEY,n integer);INSERT INTO deadlock_probe VALUES(1,0),(2,0)"); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan error, 2)
	for i := 1; i <= 2; i++ {
		first := i
		second := 3 - i
		go func() {
			var once sync.Once
			results <- h.store.runner.Do(h.ctx, func(tx pgx.Tx) error {
				if _, err := tx.Exec(h.ctx, "UPDATE deadlock_probe SET n=n+1 WHERE id=$1", first); err != nil {
					return err
				}
				once.Do(func() { ready <- struct{}{}; <-release })
				_, err := tx.Exec(h.ctx, "UPDATE deadlock_probe SET n=n+1 WHERE id=$1", second)
				return err
			})
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-ready:
		case <-time.After(3 * time.Second):
			close(release)
			t.Fatal("deadlock barrier")
		}
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if h.count(t, "SELECT count(*) FROM deadlock_probe WHERE n=2") != 2 {
		t.Fatal("deadlock retry duplicated or lost updates")
	}
}

func TestCipherPurposeKeyAndTamper(t *testing.T) {
	if _, err := (&Cipher{}).Seal("purpose", nil); !errors.Is(err, ErrCipher) {
		t.Fatal("unconfigured cipher")
	}
	c, err := NewCipher("k1", map[string][]byte{"k1": bytes.Repeat([]byte{1}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Seal("snapshot:id", []byte(`{"private":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Open("state:id", b); !errors.Is(err, ErrCipher) {
		t.Fatal("purpose mismatch")
	}
	b[len(b)-1] ^= 1
	if _, err = c.Open("snapshot:id", b); !errors.Is(err, ErrCipher) {
		t.Fatal("tamper")
	}
	rotated, err := NewCipher("k2", map[string][]byte{"k1": bytes.Repeat([]byte{1}, 32), "k2": bytes.Repeat([]byte{2}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	b, _ = c.Seal("snapshot:id", []byte(`{}`))
	out, err := rotated.Open("snapshot:id", b)
	if err != nil || string(out) != "{}" {
		t.Fatal("old key reader", err)
	}
}

func TestReplayExpiryAndCurrentOwnership(t *testing.T) {
	h := database(t, false)
	a, p := prepared(t, h, 0)
	if _, err := h.store.Place(h.ctx, p); err != nil {
		t.Fatal(err)
	}
	var scope, key, requestHash string
	if err := h.db.QueryRow(h.ctx, "SELECT scope,key,request_hash FROM idempotency_records WHERE operation_id=$1", a.OperationID).Scan(&scope, &key, &requestHash); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec(h.ctx, "UPDATE idempotency_records SET response_expires_at=clock_timestamp()-interval '1 second' WHERE operation_id=$1", a.OperationID); err != nil {
		t.Fatal(err)
	}
	ref, err := h.store.LookupAcceptance(h.ctx, scope, key, requestHash)
	if !errors.Is(err, ErrReplayExpired) || ref.OperationID != a.OperationID || ref.OrderID == nil {
		t.Fatal("expired response lost identity", err)
	}
	if _, err = h.db.Exec(h.ctx, "UPDATE orders SET scope=$1 WHERE operation_id=$2", "customer:"+uuid.NewString(), a.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err = h.store.LookupAcceptance(h.ctx, scope, key, requestHash); !errors.Is(err, ErrScope) {
		t.Fatal("old owner replay after claim", err)
	}
}

func TestPlacedSchemaPinAndPaymentCode(t *testing.T) {
	p := filepath.Join("..", "..", "..", "..", "packages", "events", "schemas", "order", "v3", "placed.event.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if hash(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))) != PlacedSchemaHash {
		t.Fatal("placed contract pin drift")
	}
	for i := 0; i < 10; i++ {
		code, err := paymentCode()
		if err != nil || len(code) != 15 || !strings.HasPrefix(code, "OMP") {
			t.Fatal("SePay reference contract", err)
		}
		for _, c := range code[3:] {
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
				t.Fatal("invalid reference alphabet")
			}
		}
	}
}

func TestAcceptanceRollbackExpiredQuoteAndRetryBudget(t *testing.T) {
	h := database(t, false)
	in := acceptanceInput(t, 0)
	in.QuoteExpiresAt = time.Now().Add(-time.Second)
	if _, err := h.store.Accept(h.ctx, in); !errors.Is(err, ErrExpired) {
		t.Fatal(err)
	}
	if h.count(t, "SELECT count(*) FROM checkout_operations") != 0 {
		t.Fatal("expired quote accepted")
	}
	_, err := h.db.Exec(h.ctx, `CREATE FUNCTION reject_accept() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION USING ERRCODE='23514',MESSAGE='synthetic fault';END$$;
 CREATE TRIGGER reject_accept BEFORE INSERT ON audit_records FOR EACH ROW EXECUTE FUNCTION reject_accept()`)
	if err != nil {
		t.Fatal(err)
	}
	in.QuoteExpiresAt = time.Now().Add(time.Minute)
	if _, err = h.store.Accept(h.ctx, in); err == nil {
		t.Fatal("accept fault ignored")
	}
	for _, table := range []string{"checkout_operations", "idempotency_records", "checkout_quote_consumptions", "checkout_submission_guards", "workflows", "jobs"} {
		if h.count(t, "SELECT count(*) FROM "+table) != 0 {
			t.Fatal("partial acceptance " + table)
		}
	}
	tries := 0
	err = h.store.runner.Do(h.ctx, func(tx pgx.Tx) error {
		tries++
		_, err := tx.Exec(h.ctx, `DO $$BEGIN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='always fail';END$$`)
		return err
	})
	if err == nil || tries != 3 {
		t.Fatal("unbounded transaction retry", tries, err)
	}
}

func TestUnknownCommitWithPrimaryUnavailableKeepsIdentity(t *testing.T) {
	h := database(t, false)
	cfg := h.db.Config()
	in := acceptanceInput(t, 0)
	fault := &ackFault{db: h.db, afterCommit: true, afterHook: func() { h.db.Close() }}
	h.store.runner.DB = fault
	if _, err := h.store.Accept(h.ctx, in); !errors.Is(err, ErrCommitUnknown) {
		t.Fatal("unavailable lookup lost UNKNOWN", err)
	}
	if fault.commits.Load() != 1 {
		t.Fatal("uncertain closure replay")
	}
	reopened, err := pgxpool.NewWithConfig(h.ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	h.store.db = reopened
	h.store.runner.DB = reopened
	old, err := h.store.LookupAcceptance(h.ctx, ownerScope(in.Owner), in.Key, in.RequestHash)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := h.store.Accept(h.ctx, in)
	if err != nil || replay.OperationID != old.OperationID {
		t.Fatal("new identity on restore connectivity", err)
	}
}

func TestLeaseExpiresDuringPlacementRollsBack(t *testing.T) {
	h := database(t, false)
	_, p := prepared(t, h, 0)
	_, err := h.db.Exec(h.ctx, `CREATE FUNCTION delay_publish() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN PERFORM pg_sleep(0.15);RETURN NEW;END$$;
 CREATE TRIGGER delay_publish BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION delay_publish();
 UPDATE jobs SET lease_until=clock_timestamp()+interval '100 milliseconds' WHERE status='LEASED'`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.store.Place(h.ctx, p); !errors.Is(err, ErrLease) {
		t.Fatal("lease expired before apply commit", err)
	}
	if h.count(t, "SELECT count(*) FROM orders") != 0 || h.count(t, "SELECT count(*) FROM outbox_events") != 0 {
		t.Fatal("stale writer partial commit")
	}
}
