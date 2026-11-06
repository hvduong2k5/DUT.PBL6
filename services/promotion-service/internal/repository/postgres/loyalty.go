package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLoyaltyIdempotencyConflict = errors.New("loyalty idempotency key was reused with a different request")
	ErrLoyaltySourceConflict      = errors.New("loyalty source already has a ledger entry")
)

type LoyaltyRepository struct{ pool *pgxpool.Pool }

func NewLoyaltyRepository(pool *pgxpool.Pool) *LoyaltyRepository {
	return &LoyaltyRepository{pool: pool}
}

type LoyaltyEntryInput struct {
	CustomerID       string
	EntryType        string
	PointsDelta      int64
	BasisAmountVND   *int64
	SourceType       string
	SourceID         string
	SourceDecisionID string
	IdempotencyKey   string
	PolicyVersion    string
	ReferenceEntryID string
}

type LoyaltyEntry struct {
	ID               string
	CustomerID       string
	EntryType        string
	PointsDelta      int64
	BasisAmountVND   *int64
	SourceType       string
	SourceID         string
	SourceDecisionID string
	IdempotencyKey   string
	PolicyVersion    string
	ReferenceEntryID string
	CreatedAt        time.Time
	Replay           bool
}

func (r *LoyaltyRepository) AppendEntry(ctx context.Context, input LoyaltyEntryInput) (LoyaltyEntry, error) {
	if err := validateLoyaltyEntry(input); err != nil {
		return LoyaltyEntry{}, err
	}
	fingerprint, err := json.Marshal(struct {
		CustomerID       string `json:"customer_id"`
		EntryType        string `json:"entry_type"`
		PointsDelta      int64  `json:"points_delta"`
		BasisAmountVND   *int64 `json:"basis_amount_vnd"`
		SourceType       string `json:"source_type"`
		SourceID         string `json:"source_id"`
		SourceDecisionID string `json:"source_decision_id"`
		PolicyVersion    string `json:"policy_version"`
		ReferenceEntryID string `json:"reference_entry_id"`
	}{input.CustomerID, input.EntryType, input.PointsDelta, input.BasisAmountVND, input.SourceType,
		input.SourceID, input.SourceDecisionID, input.PolicyVersion, input.ReferenceEntryID})
	if err != nil {
		return LoyaltyEntry{}, err
	}
	hash := sha256.Sum256(fingerprint)
	requestHash := hex.EncodeToString(hash[:])

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LoyaltyEntry{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, input.IdempotencyKey); err != nil {
		return LoyaltyEntry{}, err
	}
	existing, err := scanLoyaltyEntry(tx.QueryRow(ctx, `SELECT id::text,customer_id,entry_type,points_delta,basis_amount_vnd,source_type,source_id,COALESCE(source_decision_id,''),idempotency_key,request_hash,policy_version,COALESCE(reference_entry_id::text,''),created_at
		FROM loyalty_ledger_entries WHERE idempotency_key=$1`, input.IdempotencyKey))
	if err == nil {
		if existing.RequestHash != requestHash {
			return LoyaltyEntry{}, ErrLoyaltyIdempotencyConflict
		}
		existing.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return LoyaltyEntry{}, err
		}
		return existing.LoyaltyEntry, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return LoyaltyEntry{}, err
	}
	entry, err := scanLoyaltyEntry(tx.QueryRow(ctx, `INSERT INTO loyalty_ledger_entries
		(customer_id,entry_type,points_delta,basis_amount_vnd,source_type,source_id,source_decision_id,idempotency_key,request_hash,policy_version,reference_entry_id)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,NULLIF($11,'')::uuid)
		RETURNING id::text,customer_id,entry_type,points_delta,basis_amount_vnd,source_type,source_id,COALESCE(source_decision_id,''),idempotency_key,request_hash,policy_version,COALESCE(reference_entry_id::text,''),created_at`,
		input.CustomerID, input.EntryType, input.PointsDelta, input.BasisAmountVND, input.SourceType, input.SourceID,
		input.SourceDecisionID, input.IdempotencyKey, requestHash, input.PolicyVersion, input.ReferenceEntryID))
	if err != nil {
		if uniqueViolation(err) {
			return LoyaltyEntry{}, ErrLoyaltySourceConflict
		}
		return LoyaltyEntry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoyaltyEntry{}, err
	}
	return entry.LoyaltyEntry, nil
}

func (r *LoyaltyRepository) GetBalance(ctx context.Context, customerID string) (int64, error) {
	if strings.TrimSpace(customerID) == "" {
		return 0, errors.New("customer id is required")
	}
	var balance int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(points_delta),0)::bigint FROM loyalty_ledger_entries WHERE customer_id=$1`, customerID).Scan(&balance)
	return balance, err
}

type scannedLoyaltyEntry struct {
	LoyaltyEntry
	RequestHash string
}

func scanLoyaltyEntry(row interface{ Scan(...any) error }) (scannedLoyaltyEntry, error) {
	var entry scannedLoyaltyEntry
	err := row.Scan(&entry.ID, &entry.CustomerID, &entry.EntryType, &entry.PointsDelta, &entry.BasisAmountVND,
		&entry.SourceType, &entry.SourceID, &entry.SourceDecisionID, &entry.IdempotencyKey, &entry.RequestHash,
		&entry.PolicyVersion, &entry.ReferenceEntryID, &entry.CreatedAt)
	return entry, err
}

func validateLoyaltyEntry(input LoyaltyEntryInput) error {
	if strings.TrimSpace(input.CustomerID) == "" || strings.TrimSpace(input.SourceID) == "" ||
		strings.TrimSpace(input.PolicyVersion) == "" || input.PointsDelta == 0 || !validLoyaltyIdempotencyKey(input.IdempotencyKey) {
		return errors.New("customer, source, policy version, non-zero points and a valid idempotency key are required")
	}
	if input.BasisAmountVND != nil && *input.BasisAmountVND < 0 {
		return errors.New("basis amount must not be negative")
	}
	switch input.EntryType {
	case "PURCHASE_EARN", "REFUND_CONVERSION":
		if input.PointsDelta < 0 {
			return errors.New("credit entries require positive points")
		}
	case "REDEMPTION_DEBIT", "REVERSAL", "EXPIRATION":
		if input.PointsDelta > 0 {
			return errors.New("debit entries require negative points")
		}
	case "MANUAL_ADJUSTMENT":
	default:
		return errors.New("unsupported loyalty entry type")
	}
	switch input.SourceType {
	case "ORDER_COMPLETION", "REFUND_DECISION", "CHECKOUT_REDEMPTION", "ORDER_OR_RETURN_REVERSAL", "POINTS_EXPIRY", "ADMIN_ADJUSTMENT":
	default:
		return errors.New("unsupported loyalty source type")
	}
	return nil
}

func validLoyaltyIdempotencyKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, character := range key {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}
