package postgres

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func TestExpiryWorkerIdempotentAndDoesNotSuspend(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	if _, err := store.PublishRevision(ctx, reviewMeta("publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	worker := &application.ExpiryWorker{Store: store, PageSize: 1}
	if count, err := worker.Tick(ctx); err != nil || count != 0 {
		t.Fatal("early expiry observation", err)
	}
	store.Clock = func() time.Time { return time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC) }
	var wg sync.WaitGroup
	counts := make(chan int, 2)
	failures := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, n, err := store.SweepExpiry(ctx, "", 10)
			if err != nil {
				failures <- err
			}
			counts <- n
		}()
	}
	wg.Wait()
	close(counts)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	total := 0
	for n := range counts {
		total += n
	}
	if total != 1 {
		t.Fatalf("duplicate observation count %d", total)
	}
	for range 4 {
		if count, err := worker.Tick(ctx); err != nil || count != 0 {
			t.Fatal("worker emitted repeatedly", err)
		}
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Status != "ACTIVE" || current.Version != 5 || current.Revisions[0].Status != "PUBLISHED" {
		t.Fatal("worker suspended or rewrote profile")
	}
	var events, observations int
	if err = store.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM outbox_events),(SELECT count(*) FROM eligibility_observations)`).Scan(&events, &observations); err != nil || events != 2 || observations != 1 {
		t.Fatal("duplicate expiry side effects")
	}
	var gone *domain.GoneError
	if _, err = store.Public(ctx, p.QRCode); !errors.As(err, &gone) || gone.Reason != "CERTIFICATE_EXPIRED" {
		t.Fatal("worker/read guard mismatch", err)
	}
	if _, err = store.NewRevision(ctx, reviewMeta("replace-expired"), p.ID, domain.NewRevisionRequest{ExpectedVersion: 5, FromRevision: 1}); err != nil {
		t.Fatal("cannot revise expired certificate", err)
	}
}

func TestExpiryObservationRollsBackOnOutboxFailure(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	if _, err := store.PublishRevision(ctx, reviewMeta("publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	store.Clock = func() time.Time { return time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC) }
	if _, err := store.DB.Exec(ctx, `CREATE FUNCTION fail_expiry_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic expiry event failure'; END $$; CREATE TRIGGER fail_expiry BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION fail_expiry_test()`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SweepExpiry(ctx, "", 10); err == nil {
		t.Fatal("outbox failure accepted")
	}
	var observations int
	if err := store.DB.QueryRow(ctx, `SELECT count(*) FROM eligibility_observations`).Scan(&observations); err != nil || observations != 0 {
		t.Fatal("observation partially committed")
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Version != 4 {
		t.Fatal("expiry version partially committed")
	}
}
