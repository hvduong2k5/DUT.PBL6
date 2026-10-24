package persistence

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain/core"
)

const Cohort = "order-core-1"

var (
	ErrInput         = errors.New("INVALID_PERSISTENCE_INPUT")
	ErrConflict      = errors.New("IDEMPOTENCY_CONFLICT")
	ErrQuoteUsed     = errors.New("QUOTE_ALREADY_USED")
	ErrSubmitted     = errors.New("CHECKOUT_ALREADY_SUBMITTED")
	ErrExpired       = errors.New("QUOTE_EXPIRED")
	ErrLegacy        = errors.New("IDENTITY_BELONGS_TO_LEGACY_COHORT")
	ErrIntegrity     = errors.New("CORE_PERSISTENCE_INTEGRITY")
	ErrLease         = errors.New("STALE_WORKFLOW_LEASE")
	ErrReplayExpired = errors.New("IDEMPOTENCY_RESPONSE_EXPIRED")
	ErrScope         = errors.New("OWNER_SCOPE_DENIED")
)

type Store struct {
	db       *pgxpool.Pool
	runner   Runner
	cipher   *Cipher
	auditKey []byte
}

func NewStore(db *pgxpool.Pool, cipher *Cipher, auditKey []byte) (*Store, error) {
	if db == nil || cipher == nil || cipher.keys[cipher.current] == nil || len(auditKey) < 32 {
		return nil, ErrInput
	}
	return &Store{db, Runner{db}, cipher, append([]byte(nil), auditKey...)}, nil
}

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validHash(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}
func validID(s string) bool {
	u, e := uuid.Parse(s)
	return e == nil && u != uuid.Nil && u.String() == s
}
func ownerScope(o core.Owner) string { return strings.ToLower(o.Kind) + ":" + o.PrincipalID }

type AcceptanceInput struct {
	Owner            core.Owner
	Method           string
	Key, RequestHash string
	QuoteID, CartID  string
	CartRevision     int64
	SelectionDigest  core.SelectionHash
	QuoteHash        core.PayloadHash
	QuoteExpiresAt   time.Time
	AcceptedJSON     []byte
}
type Acceptance struct {
	OperationID    string
	PlannedOrderID string
	OrderID        *string
	Status         string
	AcceptedAt     time.Time
}

// Accept is an internal SQL boundary over an already-authorized/admitted quote.
// It performs no Redis access or stock effect. Stage5 must validate quote bytes,
// identity, cart guard and canonical terms before supplying this input.
func (s *Store) Accept(ctx context.Context, in AcceptanceInput) (Acceptance, error) {
	if in.Owner.Validate() != nil || (in.Method != "VIETQR" && in.Method != "COD") || !validID(in.QuoteID) || !validID(in.CartID) || in.CartRevision < 0 ||
		!validHash(in.RequestHash) || !validHash(string(in.SelectionDigest)) || !validHash(string(in.QuoteHash)) ||
		in.QuoteExpiresAt.IsZero() || len(in.AcceptedJSON) > 1<<20 || !json.Valid(in.AcceptedJSON) {
		return Acceptance{}, ErrInput
	}
	// Reuse the Core key/scope validation, not the legacy length-only validator.
	if _, err := core.NewCommandIdentity(ownerScope(in.Owner), "CHECKOUT", in.QuoteID, in.Key, []byte(`{}`)); err != nil {
		return Acceptance{}, ErrInput
	}
	op, planned, workflow, job := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	cipher, err := s.cipher.Seal("accepted:"+op+":"+ownerScope(in.Owner), in.AcceptedJSON)
	if err != nil {
		return Acceptance{}, err
	}
	var out Acceptance
	err = s.runner.Do(ctx, func(tx pgx.Tx) error {
		// Serializes same key even before its registry row exists. Hash collisions
		// only reduce concurrency; database unique constraints remain authority.
		if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "CHECKOUT:"+ownerScope(in.Owner)+":"+in.Key); e != nil {
			return e
		}
		existing, e := s.lookupAcceptance(ctx, tx, ownerScope(in.Owner), in.Key, in.RequestHash)
		if e == nil {
			out = existing
			return nil
		}
		if errors.Is(e, ErrReplayExpired) {
			out = existing
			return e
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var at time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&at); e != nil {
			return e
		}
		if !at.Before(in.QuoteExpiresAt) {
			return ErrExpired
		}
		_, e = tx.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,stage,input_cipher,request_hash,accepted_at,
 cohort,quote_id,cart_id,cart_revision,selection_digest,quote_hash,input_hash,input_version,owner_kind,customer_id,guest_session_id,payment_method)
 VALUES($1,$2,$3,$4,'ACCEPTED','VALIDATE',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1,$15,$16,$17,$18)`,
			op, ownerScope(in.Owner), in.Owner.PrincipalID, planned, cipher, in.RequestHash, at, Cohort, in.QuoteID, in.CartID, in.CartRevision,
			in.SelectionDigest, in.QuoteHash, hash(in.AcceptedJSON), in.Owner.Kind, in.Owner.CustomerID, in.Owner.GuestSessionID, in.Method)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO idempotency_records(scope,kind,key,request_hash,operation_id) VALUES($1,'CHECKOUT',$2,$3,$4)`, ownerScope(in.Owner), in.Key, in.RequestHash, op); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO checkout_quote_consumptions(quote_id,operation_id,owner_scope,request_hash,claimed_at) VALUES($1,$2,$3,$4,$5)`, in.QuoteID, op, ownerScope(in.Owner), in.RequestHash, at); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO checkout_submission_guards(owner_scope,cart_id,cart_revision,selection_digest,operation_id,state)
 VALUES($1,$2,$3,$4,$5,'ACTIVE') ON CONFLICT(owner_scope,cart_id,cart_revision,selection_digest)
 DO UPDATE SET operation_id=EXCLUDED.operation_id,state='ACTIVE',version=checkout_submission_guards.version+1
 WHERE checkout_submission_guards.state='RELEASED' RETURNING operation_id`, ownerScope(in.Owner), in.CartID, in.CartRevision, in.SelectionDigest, op)
		// Exec's affected count distinguishes a consumed/active fingerprint.
		if e != nil {
			return e
		}
		var held string
		if e = tx.QueryRow(ctx, `SELECT operation_id FROM checkout_submission_guards WHERE owner_scope=$1 AND cart_id=$2 AND cart_revision=$3 AND selection_digest=$4`, ownerScope(in.Owner), in.CartID, in.CartRevision, in.SelectionDigest).Scan(&held); e != nil {
			return e
		}
		if held != op {
			return ErrSubmitted
		}
		if _, e = tx.Exec(ctx, `INSERT INTO workflows(id,kind,workflow_version,aggregate_type,aggregate_id,stage,state,policy_version,stage_started_at)
 VALUES($1,'CHECKOUT',1,'CHECKOUT_OPERATION',$2,'VALIDATE','RUNNING','cart-checkout-1.0',$3)`, workflow, op, at); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO jobs(id,kind,key,stage,status,cohort,workflow_id,input_version,input_cipher,input_hash,lane)
 VALUES($1,'CHECKOUT_VALIDATE',$2,'VALIDATE','READY',$3,$4,1,$5,$6,'INTERACTIVE')`, job, "checkout:"+op+":validate", Cohort, workflow, cipher, hash(in.AcceptedJSON)); e != nil {
			return e
		}
		if e = s.audit(ctx, tx, "", in.Owner.PrincipalID, "CHECKOUT_ACCEPTED", map[string]any{"operation_id": op, "cohort": Cohort}); e != nil {
			return e
		}
		out = Acceptance{op, planned, nil, "ACCEPTED", at}
		return nil
	})
	if err == nil {
		return out, nil
	}
	if errors.Is(err, ErrReplayExpired) {
		return out, err
	}
	if errors.Is(err, ErrCommitUnknown) || unique(err) {
		lookup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		found, e := s.lookupAcceptance(lookup, s.db, ownerScope(in.Owner), in.Key, in.RequestHash)
		if e == nil {
			return found, nil
		}
		if errors.Is(e, ErrReplayExpired) {
			return found, e
		}
		if errors.Is(e, ErrConflict) || errors.Is(e, ErrLegacy) || errors.Is(e, ErrScope) {
			return Acceptance{}, e
		}
		if errors.Is(err, ErrCommitUnknown) {
			return Acceptance{}, err
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return Acceptance{}, e
		}
		var p *pgconn.PgError
		_ = errors.As(err, &p)
		if p != nil && (p.ConstraintName == "core_operation_quote" || p.ConstraintName == "checkout_quote_consumptions_pkey") {
			return Acceptance{}, ErrQuoteUsed
		}
	}
	return Acceptance{}, err
}

type rowReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Store) lookupAcceptance(ctx context.Context, q rowReader, scope, key, requestHash string) (Acceptance, error) {
	var out Acceptance
	var storedHash, cohort string
	var currentScope *string
	var expired bool
	err := q.QueryRow(ctx, `SELECT c.id,c.planned_order_id,o.id,c.status,c.accepted_at,i.request_hash,c.cohort,o.scope,
 clock_timestamp()>=i.response_expires_at AND c.status IN ('SUCCEEDED','FAILED')
 FROM idempotency_records i JOIN checkout_operations c ON c.id=i.operation_id
 LEFT JOIN orders o ON o.operation_id=c.id
 WHERE i.scope=$1 AND i.kind='CHECKOUT' AND i.key=$2`, scope, key).Scan(&out.OperationID, &out.PlannedOrderID, &out.OrderID, &out.Status, &out.AcceptedAt, &storedHash, &cohort, &currentScope, &expired)
	if err != nil {
		return out, err
	}
	if cohort != Cohort {
		return Acceptance{}, ErrLegacy
	}
	if currentScope != nil && *currentScope != scope {
		return Acceptance{}, ErrScope
	}
	if storedHash != requestHash {
		return Acceptance{}, ErrConflict
	}
	if out.Status == "SUCCEEDED" && out.OrderID == nil {
		return Acceptance{}, ErrIntegrity
	}
	if expired {
		return out, ErrReplayExpired
	}
	return out, nil
}

func (s *Store) LookupAcceptance(ctx context.Context, scope, key, requestHash string) (Acceptance, error) {
	return s.lookupAcceptance(ctx, s.db, scope, key, requestHash)
}

func unique(err error) bool { var p *pgconn.PgError; return errors.As(err, &p) && p.Code == "23505" }

func (s *Store) AcceptedInput(ctx context.Context, operationID, scope string) ([]byte, error) {
	var encrypted []byte
	var expected string
	err := s.db.QueryRow(ctx, `SELECT input_cipher,input_hash FROM checkout_operations WHERE id=$1 AND scope=$2 AND cohort=$3`, operationID, scope, Cohort).Scan(&encrypted, &expected)
	if err != nil {
		return nil, err
	}
	data, err := s.cipher.Open("accepted:"+operationID+":"+scope, encrypted)
	if err != nil {
		return nil, err
	}
	if hash(data) != expected {
		return nil, ErrIntegrity
	}
	return data, nil
}

func (s *Store) audit(ctx context.Context, tx pgx.Tx, orderID, actor, action string, details any) error {
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(440031)"); err != nil {
		return err
	}
	var prev string
	err := tx.QueryRow(ctx, "SELECT hash FROM audit_records ORDER BY sequence DESC LIMIT 1").Scan(&prev)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	id := uuid.NewString()
	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, s.auditKey)
	fmt.Fprintf(mac, "%s|%s|%s|%s|%s|%s", prev, id, orderID, actor, action, data)
	_, err = tx.Exec(ctx, `INSERT INTO audit_records(id,order_id,actor,action,details,prev_hash,hash)
 VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7)`, id, orderID, actor, action, data, prev, hex.EncodeToString(mac.Sum(nil)))
	return err
}
