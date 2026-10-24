package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	"github.com/jackc/pgx/v5"
)

const testProfile = "11111111-1111-4111-8111-111111111111"
const testActor = "abcdefab-2222-4222-8222-222222222222"

func TestFingerprint(t *testing.T) {
	a, err := Fingerprint("create", "", map[string]any{"b": 2, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Fingerprint("create", "", map[string]any{"a": 1, "b": 2})
	if a != b {
		t.Fatal("map ordering changed fingerprint")
	}
	for _, pair := range [][2]string{{"edit", ""}, {"create", "other"}} {
		other, _ := Fingerprint(pair[0], pair[1], map[string]any{"a": 1, "b": 2})
		if a == other {
			t.Fatal("action/resource absent from fingerprint")
		}
	}
	if _, err = Fingerprint("create", "", make(chan int)); err == nil {
		t.Fatal("unencodable request accepted")
	}
}

func seedCommandProfile(ctx context.Context, tx pgx.Tx) (Outcome, error) {
	_, err := tx.Exec(ctx, `INSERT INTO traceability_profiles(id,mode,product_id,sku_code,source_namespace,batch_id,qr_code)
 VALUES($1,'sandbox','44444444-4444-4444-8444-444444444444','SKU-1','fixture','batch-1','abcdefghijklmnopqrstuv')`, testProfile)
	return Outcome{ResourceID: testProfile, Status: 201, Body: json.RawMessage(`{"version":1,"id":"` + testProfile + `"}`)}, err
}

func command(t *testing.T) Command {
	t.Helper()
	fingerprint, err := Fingerprint("create", "", map[string]string{"batch_id": "batch-1"})
	if err != nil {
		t.Fatal(err)
	}
	return Command{PrincipalID: testActor, Action: "create", Key: "create-1", Fingerprint: fingerprint}
}

func TestCommandsReplayAndIsolation(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store := Commands{DB: pool}
	cmd := command(t)
	first, err := store.Run(ctx, cmd, seedCommandProfile)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.Run(ctx, cmd, func(context.Context, pgx.Tx) (Outcome, error) {
		t.Fatal("replay executed operation")
		return Outcome{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || !replay.Replayed || first.Status != 201 || string(first.Body) != string(replay.Body) {
		t.Fatalf("bad replay: %+v %+v", first, replay)
	}
	changed := cmd
	changed.Fingerprint = strings.Repeat("b", 64)
	if _, err = store.Run(ctx, changed, seedCommandProfile); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	for _, modified := range []Command{
		{PrincipalID: "33333333-3333-4333-8333-333333333333", Action: cmd.Action, Key: cmd.Key, Fingerprint: cmd.Fingerprint},
		{PrincipalID: cmd.PrincipalID, Action: "other", Key: cmd.Key, Fingerprint: cmd.Fingerprint},
	} {
		result, err := store.Run(ctx, modified, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
			return Outcome{ResourceID: testProfile, Status: 200, Body: json.RawMessage(`{"isolated":true}`)}, nil
		})
		if err != nil || result.Replayed {
			t.Fatalf("principal/action namespace: %+v %v", result, err)
		}
	}
}

func TestCommandsConcurrentRetry(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store := Commands{DB: pool}
	cmd := command(t)
	var calls atomic.Int32
	var wg sync.WaitGroup
	results := make(chan Outcome, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			local := cmd
			if i%2 == 0 {
				local.PrincipalID = strings.ToUpper(local.PrincipalID)
			}
			result, err := store.Run(ctx, local, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
				calls.Add(1)
				return seedCommandProfile(ctx, tx)
			})
			if err != nil {
				failures <- err
			} else {
				results <- result
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	close(results)
	for err := range failures {
		t.Error(err)
	}
	originals, replays := 0, 0
	for result := range results {
		if result.Replayed {
			replays++
		} else {
			originals++
		}
	}
	if calls.Load() != 1 || originals != 1 || replays != 7 {
		t.Fatalf("calls=%d originals=%d replay=%d", calls.Load(), originals, replays)
	}
}

func TestCommandsRollback(t *testing.T) {
	for _, scenario := range []string{"business_error", "invalid_outcome", "deferred_constraint"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, pool := testutil.Database(t)
			store := Commands{DB: pool}
			cmd := command(t)
			_, err := store.Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
				result, err := seedCommandProfile(ctx, tx)
				if err != nil {
					return result, err
				}
				if scenario == "business_error" {
					return result, errors.New("simulated business failure")
				}
				if scenario == "invalid_outcome" {
					result.Status = 500
					return result, nil
				}
				_, err = tx.Exec(ctx, `UPDATE traceability_profiles SET public_revision=99,ever_published=true,version=2`)
				return result, err
			})
			if err == nil {
				t.Fatal("failed command accepted")
			}
			var profiles, records int
			if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM traceability_profiles),(SELECT count(*) FROM idempotency_records)`).Scan(&profiles, &records); err != nil {
				t.Fatal(err)
			}
			if profiles != 0 || records != 0 {
				t.Fatalf("partial transaction: profiles=%d keys=%d", profiles, records)
			}
			if _, err = store.Run(ctx, cmd, seedCommandProfile); err != nil {
				t.Fatalf("retry after rollback: %v", err)
			}
		})
	}
}

func TestCommandsValidation(t *testing.T) {
	cmd := command(t)
	noop := func(context.Context, pgx.Tx) (Outcome, error) { return Outcome{}, nil }
	if _, err := (Commands{}).Run(context.Background(), cmd, noop); !errors.Is(err, ErrInvalidCommand) {
		t.Fatal(err)
	}
	ctx, pool := testutil.Database(t)
	store := Commands{DB: pool}
	for _, field := range []string{"principal", "action", "key", "fingerprint", "callback"} {
		t.Run(field, func(t *testing.T) {
			bad := cmd
			operation := noop
			switch field {
			case "principal":
				bad.PrincipalID = "not-uuid"
			case "action":
				bad.Action = "../unsafe"
			case "key":
				bad.Key = "unsafe key"
			case "fingerprint":
				bad.Fingerprint = "wrong"
			case "callback":
				operation = nil
			}
			if _, err := store.Run(ctx, bad, operation); !errors.Is(err, ErrInvalidCommand) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}
}
