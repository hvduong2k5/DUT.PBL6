package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/omamx/shipping-service/internal/repository/postgres"
	"github.com/omamx/shipping-service/migrations"
)

func TestPostgresInternalDeliveryAssignments(t *testing.T) {
	dsn := os.Getenv("SHIPPING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SHIPPING_TEST_DATABASE_URL required; CI supplies PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "shipping_assignment_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if applied, err := migrations.Apply(ctx, pool); err != nil || len(applied) != 0 {
		t.Fatalf("migration replay: %v %v", applied, err)
	}
	repo := postgres.New(pool)
	provider, _ := carrier.NewInternal(25000, 24*time.Hour)
	svc := application.NewService(repo, carrier.NewRegistry(provider, carrier.NewFake()))
	create := func(code string) *domain.Shipment {
		t.Helper()
		id := uuid.NewString()
		s, err := svc.CreateShipment(ctx, domain.CreateShipmentCommand{IdempotencyKey: "create-" + id, OrderID: id, PackageID: id, CarrierCode: code, ServiceCode: "STANDARD", Recipient: domain.Recipient{Name: "Fixture", Phone: "0900000001", Street: "Test", Ward: "Ward", Province: "Province", CountryCode: "VN"}, Parcel: domain.Parcel{WeightGrams: 500, LengthCM: 20, WidthCM: 15, HeightCM: 10}})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	command := func(s *domain.Shipment) domain.AssignInternalDelivery {
		return domain.AssignInternalDelivery{ShipmentID: s.ID, DriverID: uuid.New(), ActorID: uuid.New(), Reason: "Initial dispatch", IdempotencyKey: "assign-" + uuid.NewString()}
	}

	t.Run("driver progress atomic outbox revisions replay and return", func(t *testing.T) {
		prepare := func() (*domain.Shipment, domain.ProgressInternalDelivery) {
			s := create(carrier.InternalCode)
			if _, err := pool.Exec(ctx, `UPDATE shipments SET event_version=2,ready_generation=1,seal_code='INTERNAL-SEAL' WHERE id=$1`, s.ID); err != nil {
				t.Fatal(err)
			}
			a := command(s)
			if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
				t.Fatal(err)
			}
			c := domain.ProgressInternalDelivery{ShipmentID: s.ID, ActorID: a.DriverID, AssignmentVersion: 1, ExpectedRevision: s.Revision, Status: domain.StatusInTransit, Reason: "Started route", IdempotencyKey: "progress-" + uuid.NewString()}
			if _, err := svc.ProgressInternalDelivery(ctx, c); !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("unaccepted: %v", err)
			}
			if _, err := svc.AcceptInternalDelivery(ctx, domain.AcceptInternalDelivery{ShipmentID: s.ID, ActorID: a.DriverID, ExpectedVersion: 1, IdempotencyKey: "accept-" + uuid.NewString()}); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.ProgressInternalDelivery(ctx, c); !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("before handover: %v", err)
			}
			s, err := svc.ConfirmHandover(ctx, s.ID, "handover-"+uuid.NewString())
			if err != nil {
				t.Fatal(err)
			}
			c.ExpectedRevision = s.Revision
			return s, c
		}
		s, c := prepare()
		wrong := c
		wrong.ActorID = uuid.New()
		if _, err := svc.ProgressInternalDelivery(ctx, wrong); !errors.Is(err, domain.ErrDriverNotAssigned) {
			t.Fatalf("wrong actor: %v", err)
		}
		stale := c
		stale.AssignmentVersion++
		if _, err := svc.ProgressInternalDelivery(ctx, stale); !errors.Is(err, domain.ErrConcurrentUpdate) {
			t.Fatalf("stale assignment: %v", err)
		}
		first, err := svc.ProgressInternalDelivery(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		for _, state := range []domain.ShipmentStatus{domain.StatusOutForDelivery, domain.StatusDeliveryFailed, domain.StatusReturning, domain.StatusReturned} {
			current, err := repo.GetByID(ctx, s.ID)
			if err != nil {
				t.Fatal(err)
			}
			c.ExpectedRevision = current.Revision
			c.Status = state
			c.IdempotencyKey = "progress-" + uuid.NewString()
			if _, err := svc.ProgressInternalDelivery(ctx, c); err != nil {
				t.Fatal(err)
			}
		}
		original := c
		original.Status = domain.StatusInTransit
		original.ExpectedRevision = first.Revision - 1
		// Replay uses the original key/payload, not the latest receipt.
		var originalKey string
		if err := pool.QueryRow(ctx, `SELECT idempotency_key FROM internal_delivery_progress_events WHERE id=$1`, first.EventID).Scan(&originalKey); err != nil {
			t.Fatal(err)
		}
		original.IdempotencyKey = originalKey
		replay, err := svc.ProgressInternalDelivery(ctx, original)
		if err != nil || *replay != *first {
			t.Fatalf("replay: %+v %v", replay, err)
		}
		original.Reason = "Different"
		if _, err := svc.ProgressInternalDelivery(ctx, original); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("changed replay: %v", err)
		}
		var events, tracking, outbox int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM internal_delivery_progress_events WHERE shipment_id=$1`, s.ID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM tracking_events WHERE shipment_id=$1 AND carrier_code='INTERNAL'`, s.ID).Scan(&tracking); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events o JOIN internal_delivery_progress_events p ON o.id=p.id WHERE p.shipment_id=$1 AND o.topic='shipping.events.v2'`, s.ID).Scan(&outbox); err != nil {
			t.Fatal(err)
		}
		if events != 5 || tracking != 5 || outbox != 5 {
			t.Fatalf("atomic counts %d/%d/%d", events, tracking, outbox)
		}
		// Outbox failure must roll back shipment, timeline, audit and proof.
		s2, c2 := prepare()
		if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_internal_progress() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type LIKE '%status-changed%' THEN RAISE EXCEPTION 'fixture outbox unavailable'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_internal_progress BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION reject_internal_progress()`); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ProgressInternalDelivery(ctx, c2); err == nil {
			t.Fatal("outbox failure accepted")
		}
		if _, err := pool.Exec(ctx, `DROP TRIGGER reject_internal_progress ON outbox_events; DROP FUNCTION reject_internal_progress()`); err != nil {
			t.Fatal(err)
		}
		unchanged, err := repo.GetByID(ctx, s2.ID)
		if err != nil || unchanged.Revision != s2.Revision || unchanged.Status != s2.Status {
			t.Fatal("outbox failure changed shipment")
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM tracking_events WHERE shipment_id=$1`, s2.ID).Scan(&tracking); err != nil || tracking != 0 {
			t.Fatal("outbox failure left tracking")
		}
		if _, err := svc.ProgressInternalDelivery(ctx, c2); err != nil {
			t.Fatalf("retry after outage: %v", err)
		}
		current, err := repo.GetByID(ctx, s2.ID)
		if err != nil {
			t.Fatal(err)
		}
		c2.ExpectedRevision = current.Revision
		c2.Status = domain.StatusOutForDelivery
		c2.IdempotencyKey = "out-for-delivery-" + uuid.NewString()
		if _, err := svc.ProgressInternalDelivery(ctx, c2); err != nil {
			t.Fatal(err)
		}
		current, err = repo.GetByID(ctx, s2.ID)
		if err != nil {
			t.Fatal(err)
		}
		c2.ExpectedRevision = current.Revision
		c2.Status = domain.StatusDelivered
		c2.IdempotencyKey = "delivered-" + uuid.NewString()
		result, err := svc.ProgressInternalDelivery(ctx, c2)
		if err != nil {
			t.Fatal(err)
		}
		proof, err := repo.GetRevision(ctx, s2.ID, result.Revision)
		if err != nil || proof.Kind != domain.RevisionKindDelivered || proof.SettlementVerified {
			t.Fatalf("delivered proof: %+v %v", proof, err)
		}
		var eventType, carrierCode, resourceVersion string
		if err := pool.QueryRow(ctx, `SELECT payload->>'type',payload->'data'->>'carrier_code',payload->'data'->>'resource_version' FROM outbox_events WHERE id=$1`, result.EventID).Scan(&eventType, &carrierCode, &resourceVersion); err != nil || eventType != "vn.omama.shipping.shipment.delivered.v2" || carrierCode != "INTERNAL" || resourceVersion != fmt.Sprint(result.Revision) {
			t.Fatalf("Order wire mismatch: %s %s %s %v", eventType, carrierCode, resourceVersion, err)
		}
		_, race := prepare()
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				cmd := race
				cmd.IdempotencyKey = "progress-race-" + uuid.NewString()
				_, err := svc.ProgressInternalDelivery(ctx, cmd)
				results <- err
			}()
		}
		wins := 0
		for i := 0; i < 2; i++ {
			err := <-results
			if err == nil {
				wins++
			} else if !errors.Is(err, domain.ErrConcurrentUpdate) {
				t.Fatalf("concurrent progress: %v", err)
			}
		}
		if wins != 1 {
			t.Fatalf("concurrent progress winners: %d", wins)
		}
		c2.ExpectedRevision = result.Revision
		c2.Status = domain.StatusReturning
		c2.IdempotencyKey = "after-delivery-" + uuid.NewString()
		if _, err := svc.ProgressInternalDelivery(ctx, c2); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("terminal guard: %v", err)
		}
	})

	t.Run("acceptance guards and concurrent winner", func(t *testing.T) {
		for _, code := range []string{carrier.InternalCode, "FAKE"} {
			s := create(code)
			_, err := svc.AcceptInternalDelivery(ctx, domain.AcceptInternalDelivery{ShipmentID: s.ID, ActorID: uuid.New(), ExpectedVersion: 1, IdempotencyKey: "unassigned-" + uuid.NewString()})
			want := domain.ErrAssignmentNotFound
			if code == "FAKE" {
				want = domain.ErrInvalidShipment
			}
			if !errors.Is(err, want) {
				t.Fatalf("%s: %v", code, err)
			}
		}
		s := create(carrier.InternalCode)
		a := command(s)
		if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
			t.Fatal(err)
		}
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				_, err := svc.AcceptInternalDelivery(ctx, domain.AcceptInternalDelivery{ShipmentID: s.ID, ActorID: a.DriverID, ExpectedVersion: 1, IdempotencyKey: "race-" + uuid.NewString()})
				results <- err
			}()
		}
		wins := 0
		for i := 0; i < 2; i++ {
			err := <-results
			if err == nil {
				wins++
			} else if !errors.Is(err, domain.ErrIdempotencyConflict) {
				t.Fatal(err)
			}
		}
		if wins != 1 {
			t.Fatalf("acceptance winners: %d", wins)
		}
	})

	t.Run("accept assigned delivery and invalidate on reassignment", func(t *testing.T) {
		s := create(carrier.InternalCode)
		a := command(s)
		if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
			t.Fatal(err)
		}
		c := domain.AcceptInternalDelivery{ShipmentID: s.ID, ActorID: a.DriverID, ExpectedVersion: 1, IdempotencyKey: "accept-" + uuid.NewString()}
		wrong := c
		wrong.ActorID = uuid.New()
		if _, err := svc.AcceptInternalDelivery(ctx, wrong); !errors.Is(err, domain.ErrDriverNotAssigned) {
			t.Fatalf("wrong driver: %v", err)
		}
		stale := c
		stale.ExpectedVersion = 2
		if _, err := svc.AcceptInternalDelivery(ctx, stale); !errors.Is(err, domain.ErrConcurrentUpdate) {
			t.Fatalf("stale: %v", err)
		}
		first, err := svc.AcceptInternalDelivery(ctx, c)
		if err != nil || first.AcceptedAt.IsZero() {
			t.Fatalf("accept: %+v %v", first, err)
		}
		replay, err := svc.AcceptInternalDelivery(ctx, c)
		if err != nil || *first != *replay {
			t.Fatalf("retry: %+v %v", replay, err)
		}
		changed := c
		changed.ExpectedVersion = 2
		if _, err := svc.AcceptInternalDelivery(ctx, changed); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("changed payload: %v", err)
		}
		extra := c
		extra.IdempotencyKey = "accept-again-" + uuid.NewString()
		if _, err := svc.AcceptInternalDelivery(ctx, extra); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("double accept: %v", err)
		}
		a.ExpectedVersion = 1
		a.DriverID = uuid.New()
		a.IdempotencyKey = "reassign-" + uuid.NewString()
		if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AcceptInternalDelivery(ctx, extra); !errors.Is(err, domain.ErrDriverNotAssigned) {
			t.Fatalf("old driver: %v", err)
		}
		c.ActorID = a.DriverID
		c.ExpectedVersion = 2
		c.IdempotencyKey = "accept-new-" + uuid.NewString()
		if _, err := svc.AcceptInternalDelivery(ctx, c); err != nil {
			t.Fatal(err)
		}
		current, err := repo.GetByID(ctx, s.ID)
		if err != nil || current.Status != s.Status || current.Revision != s.Revision {
			t.Fatal("acceptance changed shipment proof")
		}
		if _, err := svc.ConfirmHandover(ctx, s.ID, "handover-"+uuid.NewString()); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AcceptInternalDelivery(ctx, c); err != nil {
			t.Fatalf("retry after handover: %v", err)
		}
		c.IdempotencyKey = "late-accept-" + uuid.NewString()
		if _, err := svc.AcceptInternalDelivery(ctx, c); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("late accept: %v", err)
		}
	})

	t.Run("assign reassign replay and audit", func(t *testing.T) {
		s := create(carrier.InternalCode)
		c := command(s)
		if _, err := svc.GetInternalDeliveryAssignment(ctx, s.ID); !errors.Is(err, domain.ErrAssignmentNotFound) {
			t.Fatalf("unassigned: %v", err)
		}
		first, err := svc.AssignInternalDelivery(ctx, c)
		if err != nil || first.Version != 1 {
			t.Fatalf("initial: %+v %v", first, err)
		}
		change := c
		change.DriverID = uuid.New()
		change.IdempotencyKey = "reassign-" + uuid.NewString()
		change.ExpectedVersion = 1
		change.Reason = "Driver unavailable"
		second, err := svc.AssignInternalDelivery(ctx, change)
		if err != nil || second.Version != 2 {
			t.Fatalf("reassign: %+v %v", second, err)
		}
		// Replay must return the first immutable result, not the new assignee.
		replay, err := svc.AssignInternalDelivery(ctx, c)
		if err != nil || *replay != *first {
			t.Fatalf("replay changed original: %+v %v", replay, err)
		}
		current, err := svc.GetInternalDeliveryAssignment(ctx, s.ID)
		if err != nil || *current != *second {
			t.Fatalf("current: %+v %v", current, err)
		}
		changed := c
		changed.Reason = "changed request"
		if _, err := svc.AssignInternalDelivery(ctx, changed); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("changed key payload: %v", err)
		}
		stale := c
		stale.IdempotencyKey = "stale-" + uuid.NewString()
		if _, err := svc.AssignInternalDelivery(ctx, stale); !errors.Is(err, domain.ErrConcurrentUpdate) {
			t.Fatalf("stale: %v", err)
		}
		unchanged := change
		unchanged.ExpectedVersion = 2
		unchanged.IdempotencyKey = "same-driver-" + uuid.NewString()
		if _, err := svc.AssignInternalDelivery(ctx, unchanged); !errors.Is(err, domain.ErrInvalidShipment) {
			t.Fatalf("no-op assignment: %v", err)
		}
		history, err := repo.InternalDeliveryAssignmentHistory(ctx, s.ID)
		if err != nil || len(history) != 2 || history[0].PreviousDriverID != nil || history[1].PreviousDriverID == nil || *history[1].PreviousDriverID != first.DriverID {
			t.Fatalf("audit: %+v %v", history, err)
		}
		latest, err := repo.GetByID(ctx, s.ID)
		if err != nil || latest.Status != s.Status || latest.Revision != s.Revision {
			t.Fatal("assignment mutated shipment proof revision")
		}
		if _, err := svc.ConfirmHandover(ctx, s.ID, "handover-"+uuid.NewString()); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AssignInternalDelivery(ctx, c); err != nil {
			t.Fatalf("original retry after handover: %v", err)
		}
		change.ExpectedVersion = 2
		change.DriverID = uuid.New()
		change.IdempotencyKey = "after-handover-" + uuid.NewString()
		if _, err := svc.AssignInternalDelivery(ctx, change); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("assigned after handover: %v", err)
		}
	})
	t.Run("concurrent assignment has one winner", func(t *testing.T) {
		s := create(carrier.InternalCode)
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			c := command(s)
			wg.Add(1)
			go func() { defer wg.Done(); <-start; _, err := svc.AssignInternalDelivery(ctx, c); results <- err }()
		}
		close(start)
		wg.Wait()
		close(results)
		ok, conflicts := 0, 0
		for err := range results {
			if err == nil {
				ok++
			} else if errors.Is(err, domain.ErrConcurrentUpdate) {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if ok != 1 || conflicts != 1 {
			t.Fatalf("winners=%d conflicts=%d", ok, conflicts)
		}
		history, err := repo.InternalDeliveryAssignmentHistory(ctx, s.ID)
		if err != nil || len(history) != 1 {
			t.Fatalf("duplicate audit: %+v %v", history, err)
		}
	})
	t.Run("other carriers COD and missing targets rejected", func(t *testing.T) {
		fake := create(carrier.FakeCode)
		if _, err := svc.AssignInternalDelivery(ctx, command(fake)); !errors.Is(err, domain.ErrInvalidShipment) {
			t.Fatalf("assigned FAKE: %v", err)
		}
		s := create(carrier.InternalCode)
		if _, err := pool.Exec(ctx, `UPDATE shipments SET carrier_code='GHN' WHERE id=$1`, s.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AssignInternalDelivery(ctx, command(s)); !errors.Is(err, domain.ErrInvalidShipment) {
			t.Fatalf("assigned GHN: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE shipments SET carrier_code='INTERNAL',cod_amount_vnd=100 WHERE id=$1`, s.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AssignInternalDelivery(ctx, command(s)); !errors.Is(err, domain.ErrInvalidShipment) {
			t.Fatalf("assigned COD: %v", err)
		}
		missing := command(s)
		missing.ShipmentID = uuid.New()
		if _, err := svc.AssignInternalDelivery(ctx, missing); !errors.Is(err, domain.ErrShipmentNotFound) {
			t.Fatalf("missing: %v", err)
		}
	})
	t.Run("key cannot be reused on another shipment", func(t *testing.T) {
		first, second := create(carrier.InternalCode), create(carrier.InternalCode)
		c := command(first)
		if _, err := svc.AssignInternalDelivery(ctx, c); err != nil {
			t.Fatal(err)
		}
		c.ShipmentID = second.ID
		if _, err := svc.AssignInternalDelivery(ctx, c); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("cross-target key: %v", err)
		}
		if _, err := repo.GetInternalDeliveryAssignment(ctx, second.ID); !errors.Is(err, domain.ErrAssignmentNotFound) {
			t.Fatal("conflict wrote assignment")
		}
	})
	t.Run("atomic audit rollback on persistence failure", func(t *testing.T) {
		s := create(carrier.InternalCode)
		c := command(s)
		_, err := pool.Exec(ctx, `CREATE FUNCTION reject_assignment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture assignment failure'; END $$;
		CREATE TRIGGER reject_assignment BEFORE INSERT ON internal_deliveries FOR EACH ROW EXECUTE FUNCTION reject_assignment()`)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(ctx, `DROP TRIGGER reject_assignment ON internal_deliveries; DROP FUNCTION reject_assignment()`)
		if _, err := svc.AssignInternalDelivery(ctx, c); err == nil {
			t.Fatal("injected failure ignored")
		}
		history, err := repo.InternalDeliveryAssignmentHistory(ctx, s.ID)
		if err != nil || len(history) != 0 {
			t.Fatalf("partial audit committed: %+v %v", history, err)
		}
		if _, err := repo.GetInternalDeliveryAssignment(ctx, s.ID); !errors.Is(err, domain.ErrAssignmentNotFound) {
			t.Fatal("partial assignment committed")
		}
	})
}
