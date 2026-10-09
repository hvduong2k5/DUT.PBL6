package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
)

func claimIdempotency(
	ctx context.Context,
	tx pgx.Tx,
	actorID uuid.UUID,
	operation string,
	key string,
	requestHash string,
	expiresAt time.Time,
) (resourceID string, created bool, err error) {
	if _, err := tx.Exec(ctx, `
		DELETE FROM idempotency_records
		WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3 AND expires_at <= NOW()
	`, actorID, operation, key); err != nil {
		return "", false, fmt.Errorf("expire idempotency key: %w", err)
	}

	var recordID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO idempotency_records (
			id, actor_id, operation, idempotency_key, request_hash, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (actor_id, operation, idempotency_key) DO NOTHING
		RETURNING id
	`, idgen.New(), actorID, operation, key, requestHash, expiresAt).Scan(&recordID)
	if err == nil {
		return "", true, nil
	}
	if err != pgx.ErrNoRows {
		return "", false, fmt.Errorf("claim idempotency key: %w", err)
	}

	var storedHash string
	err = tx.QueryRow(ctx, `
		SELECT request_hash, COALESCE(resource_id, '')
		FROM idempotency_records
		WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3
	`, actorID, operation, key).Scan(&storedHash, &resourceID)
	if err != nil {
		return "", false, fmt.Errorf("read idempotency key: %w", err)
	}
	if storedHash != requestHash {
		return "", false, domain.ErrIdempotencyConflict
	}
	if resourceID == "" {
		return "", false, fmt.Errorf("%w: idempotent operation has no resource", domain.ErrConflict)
	}
	return resourceID, false, nil
}

func completeIdempotency(
	ctx context.Context,
	tx pgx.Tx,
	actorID uuid.UUID,
	operation string,
	key string,
	resourceID string,
	status int,
) error {
	commandTag, err := tx.Exec(ctx, `
		UPDATE idempotency_records
		SET resource_id = $4, response_status = $5
		WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3
	`, actorID, operation, key, resourceID, status)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("%w: idempotency record disappeared", domain.ErrConflict)
	}
	return nil
}
