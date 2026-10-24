// Package persistence implements the opt-in order-core-1 PostgreSQL boundary.
// Its Payment slice is explicitly enabled for TEST; target checkout remains gated.
package persistence

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type Runner struct{ DB Beginner }

var ErrCommitUnknown = errors.New("SQL_COMMIT_OUTCOME_UNKNOWN")

type CommitUnknown struct{ Cause error }

func (e *CommitUnknown) Error() string        { return ErrCommitUnknown.Error() }
func (e *CommitUnknown) Unwrap() error        { return e.Cause }
func (e *CommitUnknown) Is(target error) bool { return target == ErrCommitUnknown }

func retrySQL(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && (p.Code == "40P01" || p.Code == "40001")
}

// Do retries a whole SQL-only closure, never a remote effect. Generate business
// identities outside the closure; callers resolve ambiguous COMMIT by lookup.
func (r Runner) Do(ctx context.Context, fn func(pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		err := r.attempt(ctx, fn)
		if err == nil || !retrySQL(err) || attempt == 2 {
			return err
		}
		delay := time.Duration(rand.Int63n(int64(25*time.Millisecond) << attempt))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	panic("unreachable transaction attempts")
}

func (r Runner) attempt(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='3s'"); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err == nil || retrySQL(err) || errors.Is(err, pgx.ErrTxCommitRollback) {
		return err
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && len(p.Code) >= 2 && p.Code[:2] == "23" {
		return err
	}
	// Do not replay the closure after an ACK/connection/cancellation failure.
	return &CommitUnknown{Cause: err}
}
