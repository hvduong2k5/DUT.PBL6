package postgres

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func pendingSetup(t *testing.T) (context.Context, *Profiles, domain.Profile, *Outbox) {
	t.Helper()
	ctx, store, p, snapshot := approvedSetup(t)
	if _, err := store.PublishRevision(ctx, reviewMeta("publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	queue, err := NewOutbox(store.DB, OutboxOptions{Lease: 10 * time.Second, MaxAttempts: 2, BaseRetry: time.Second, MaxRetry: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, store, p, queue
}

func TestOutboxLeaseRetryAndProfileOrdering(t *testing.T) {
	ctx, store, p, queue := pendingSetup(t)
	if _, err := store.SetStatus(ctx, reviewMeta("suspend"), p.ID, "SUSPENDED", domain.RevisionCommand{ExpectedVersion: 4, Reason: "Test suspension"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	first, found, err := queue.Claim(ctx)
	if err != nil || !found || first.Version != 4 || first.Attempts != 1 {
		t.Fatalf("first claim: %+v %v", first, err)
	}
	if _, found, err = queue.Claim(ctx); err != nil || found {
		t.Fatal("later event overtook leased predecessor")
	}
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, first.ID); err != nil {
		t.Fatal(err)
	}
	retry, found, err := queue.Claim(ctx)
	if err != nil || !found || retry.ID != first.ID || retry.LeaseToken == first.LeaseToken || retry.Attempts != 2 || string(retry.Payload) != string(first.Payload) {
		t.Fatal("lease recovery changed event", err)
	}
	if err = queue.Ack(ctx, first); !errors.Is(err, domain.ErrLeaseLost) {
		t.Fatal("stale lease ack accepted", err)
	}
	if err = queue.Ack(ctx, retry); err != nil {
		t.Fatal(err)
	}
	second, found, err := queue.Claim(ctx)
	if err != nil || !found || second.Version != 5 {
		t.Fatal("successor not released", err)
	}
	if err = queue.Fail(ctx, second, "secret://must-not-log"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatal("unsafe error code recorded")
	}
	if err = queue.Fail(ctx, second, "BROKER_PUBLISH_FAILED"); err != nil {
		t.Fatal(err)
	}
	if _, found, err = queue.Claim(ctx); err != nil || found {
		t.Fatal("retry ignored backoff")
	}
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp() WHERE id=$1`, second.ID); err != nil {
		t.Fatal(err)
	}
	second, found, err = queue.Claim(ctx)
	if err != nil || !found || second.Attempts != 2 {
		t.Fatal("retry missing", err)
	}
	if err = queue.Fail(ctx, second, "BROKER_PUBLISH_FAILED"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.SetStatus(ctx, reviewMeta("archive"), p.ID, "ARCHIVED", domain.RevisionCommand{ExpectedVersion: 5, Reason: "Archive test"}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET retry_at=clock_timestamp() WHERE status='PENDING'`); err != nil {
		t.Fatal(err)
	}
	if _, found, err = queue.Claim(ctx); err != nil || found {
		t.Fatal("successor overtook FAILED predecessor")
	}
	stats, err := queue.Stats(ctx)
	if err != nil || stats.Pending != 1 || stats.Failed != 1 || stats.OldestPendingSeconds < 0 {
		t.Fatalf("stats: %+v %v", stats, err)
	}
	var code string
	if err = store.DB.QueryRow(ctx, `SELECT error_code FROM outbox_events WHERE status='FAILED'`).Scan(&code); err != nil || strings.Contains(code, "secret") {
		t.Fatal("unsafe persisted error")
	}
}

func TestOutboxConcurrentClaimAndExhaustedCrash(t *testing.T) {
	ctx, store, _, queue := pendingSetup(t)
	var wg sync.WaitGroup
	messages := make(chan domain.OutboxMessage, 8)
	failures := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, found, err := queue.Claim(ctx)
			if err != nil {
				failures <- err
			}
			if found {
				messages <- m
			}
		}()
	}
	wg.Wait()
	close(messages)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	all := []domain.OutboxMessage{}
	for m := range messages {
		all = append(all, m)
	}
	if len(all) != 1 {
		t.Fatalf("concurrent claims: %d", len(all))
	}
	m := all[0]
	if _, err := store.DB.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, m.ID); err != nil {
		t.Fatal(err)
	}
	retry, found, err := queue.Claim(ctx)
	if err != nil || !found || retry.Attempts != 2 {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, err = queue.Claim(ctx); err != nil || found {
		t.Fatal("exhausted crash re-claimed")
	}
	stats, err := queue.Stats(ctx)
	if err != nil || stats.Failed != 1 || stats.Pending != 0 {
		t.Fatal("crashed final attempt stayed invisible", err)
	}
}
