package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
)

type ClaimRepository struct {
	pool *pgxpool.Pool
}

func NewClaimRepository(pool *pgxpool.Pool) *ClaimRepository {
	return &ClaimRepository{pool: pool}
}

// CreateClaim registers a new guest order claim with status 'VERIFIED'.
// If the order is already VERIFIED, returns domain.ErrOrderAlreadyClaimed.
func (r *ClaimRepository) CreateClaim(ctx context.Context, customerID uuid.UUID, orderID, phoneNumber string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	query := `
		INSERT INTO guest_order_claims (id, customer_id, order_id, phone_number, claim_status, claimed_at)
		VALUES ($1, $2, $3, $4, 'VERIFIED', $5)
	`
	_, err = tx.Exec(ctx, query, uuid.New(), customerID, orderID, phoneNumber, time.Now().UTC())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Check for unique violation (Postgres error code 23505) on uq_order_claim_active
			if pgErr.Code == "23505" && pgErr.ConstraintName == "uq_order_claim_active" {
				return domain.ErrOrderAlreadyClaimed
			}
		}
		return err
	}
	if err = insertEvent(ctx, tx, "GuestOrderClaim", customerID.String(), "vn.omama.profile.guest_order_claimed.v1", map[string]any{"customer_id": customerID, "order_id": orderID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RevokeClaim updates an active claim's status to 'REVOKED', freeing the order_id for reclamation.
func (r *ClaimRepository) RevokeClaim(ctx context.Context, orderID string) error {
	query := `
		UPDATE guest_order_claims
		SET claim_status = 'REVOKED'
		WHERE order_id = $1 AND claim_status = 'VERIFIED'
	`
	tag, err := r.pool.Exec(ctx, query, orderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrClaimNotFound
	}
	return nil
}

// GetActiveClaim retrieves active claim for given order_id.
func (r *ClaimRepository) GetActiveClaim(ctx context.Context, orderID string) (*domain.GuestOrderClaim, error) {
	query := `
		SELECT id, customer_id, order_id, phone_number, claim_status, claimed_at
		FROM guest_order_claims
		WHERE order_id = $1 AND claim_status = 'VERIFIED'
	`
	var c domain.GuestOrderClaim
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&c.ID, &c.CustomerID, &c.OrderID, &c.PhoneNumber, &c.ClaimStatus, &c.ClaimedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClaimNotFound
		}
		return nil, err
	}
	return &c, nil
}
