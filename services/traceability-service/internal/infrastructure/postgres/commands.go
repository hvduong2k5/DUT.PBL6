package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

var (
	ErrIdempotencyConflict = domain.ErrIdempotencyConflict
	ErrInvalidCommand      = domain.ErrInvalidCommand
	ErrInvalidOutcome      = errors.New("invalid successful command result")
	keyPattern             = regexp.MustCompile(`^[A-Za-z0-9._~-]{1,128}$`)
	fingerprintPattern     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	actionPattern          = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,99}$`)
)

type Beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// Commands owns local transaction and dedup; caller must authorize BEFORE every
// Run, including replay. Operations must use the supplied tx for all writes.
type Commands struct{ DB Beginner }

// Lookup is a preflight optimization for commands needing network verification.
// A miss is never a reservation: Run must still recheck under the advisory lock.
func (s Commands) Lookup(ctx context.Context, cmd Command) (Outcome, bool, error) {
	if s.DB == nil || !domain.ValidUUID(cmd.PrincipalID) || !actionPattern.MatchString(cmd.Action) || !keyPattern.MatchString(cmd.Key) || !fingerprintPattern.MatchString(cmd.Fingerprint) {
		return Outcome{}, false, ErrInvalidCommand
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Outcome{}, false, err
	}
	defer func() {
		clean, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(clean)
	}()
	var result Outcome
	var fingerprint string
	err = tx.QueryRow(ctx, `SELECT fingerprint,status_code,response,resource_id::text FROM idempotency_records WHERE principal_id=$1 AND action=$2 AND key=$3`, cmd.PrincipalID, cmd.Action, cmd.Key).Scan(&fingerprint, &result.Status, &result.Body, &result.ResourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Outcome{}, false, nil
	}
	if err != nil {
		return Outcome{}, false, err
	}
	if fingerprint != cmd.Fingerprint {
		return Outcome{}, false, ErrIdempotencyConflict
	}
	if err = tx.Commit(ctx); err != nil {
		return Outcome{}, false, err
	}
	result.Replayed = true
	return result, true, nil
}

type Command struct {
	PrincipalID string
	Action      string
	Key         string
	Fingerprint string
}

type Outcome struct {
	ResourceID string
	Status     int
	Body       json.RawMessage
	Replayed   bool
}

// Fingerprint hashes the typed, decoded request, action and resource, rather
// than raw request whitespace or headers. Typed requests must reject unknown
// fields before reaching this function.
func Fingerprint(action, resource string, request any) (string, error) {
	data, err := json.Marshal(struct {
		Action   string `json:"action"`
		Resource string `json:"resource"`
		Request  any    `json:"request"`
	}{action, resource, request})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s Commands) Run(ctx context.Context, cmd Command, operation func(context.Context, pgx.Tx) (Outcome, error)) (Outcome, error) {
	if s.DB == nil || operation == nil || !domain.ValidUUID(cmd.PrincipalID) || !actionPattern.MatchString(cmd.Action) || !keyPattern.MatchString(cmd.Key) || !fingerprintPattern.MatchString(cmd.Fingerprint) {
		return Outcome{}, ErrInvalidCommand
	}
	// UUID is case insensitive in PostgreSQL; advisory-lock namespace must be too.
	cmd.PrincipalID = strings.ToLower(cmd.PrincipalID)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Outcome{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	identity, _ := json.Marshal([]string{cmd.PrincipalID, cmd.Action, cmd.Key})
	lock := sha256.Sum256(identity)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(binary.BigEndian.Uint64(lock[:8]))); err != nil {
		return Outcome{}, err
	}
	var priorFingerprint string
	var result Outcome
	err = tx.QueryRow(ctx, `SELECT fingerprint,status_code,response,resource_id::text FROM idempotency_records WHERE principal_id=$1 AND action=$2 AND key=$3`, cmd.PrincipalID, cmd.Action, cmd.Key).Scan(&priorFingerprint, &result.Status, &result.Body, &result.ResourceID)
	if err == nil {
		if priorFingerprint != cmd.Fingerprint {
			return Outcome{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return Outcome{}, err
		}
		result.Replayed = true
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Outcome{}, err
	}
	result, err = operation(ctx, tx)
	if err != nil {
		return Outcome{}, err
	}
	if !domain.ValidUUID(result.ResourceID) || result.Status < 200 || result.Status > 299 || !json.Valid(result.Body) {
		return Outcome{}, ErrInvalidOutcome
	}
	// Return the same JSONB representation on first success and later replays.
	err = tx.QueryRow(ctx, `INSERT INTO idempotency_records(principal_id,action,key,fingerprint,status_code,response,resource_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING response`, cmd.PrincipalID, cmd.Action, cmd.Key, cmd.Fingerprint, result.Status, result.Body, result.ResourceID).Scan(&result.Body)
	if err != nil {
		return Outcome{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Outcome{}, err
	}
	result.Replayed = false
	return result, nil
}
