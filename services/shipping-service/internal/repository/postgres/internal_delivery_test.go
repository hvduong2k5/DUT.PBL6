package postgres_test

import (
	"context"
	"errors"
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
