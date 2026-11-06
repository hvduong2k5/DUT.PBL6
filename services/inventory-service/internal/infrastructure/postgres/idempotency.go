package postgres

import (
	"context"
	"database/sql"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
)

// PostgresIdempotencyRepository implements port.IdempotencyRepository.
type PostgresIdempotencyRepository struct {
	db *sql.DB
}

// NewPostgresIdempotencyRepository creates a new PostgresIdempotencyRepository.
func NewPostgresIdempotencyRepository(db *sql.DB) *PostgresIdempotencyRepository {
	return &PostgresIdempotencyRepository{db: db}
}

func (r *PostgresIdempotencyRepository) getExecutor(tx port.Transaction) queryExecutor {
	if tx != nil {
		if sqlTx, ok := tx.(*SQLTransaction); ok && sqlTx.Tx != nil {
			return sqlTx.Tx
		}
	}
	return r.db
}

// CheckOrSet attempts to insert an idempotency key.
// Returns true if the key was newly inserted (first execution).
// Returns false if the key already exists (duplicate request).
func (r *PostgresIdempotencyRepository) CheckOrSet(ctx context.Context, tx port.Transaction, key string) (bool, error) {
	query := `INSERT INTO idempotency_keys (key_name, created_at)
              VALUES ($1, $2)
              ON CONFLICT (key_name) DO NOTHING`
	res, err := r.getExecutor(tx).ExecContext(ctx, query, key, time.Now())
	if err != nil {
		return false, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
