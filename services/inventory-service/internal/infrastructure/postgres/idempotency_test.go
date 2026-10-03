package postgres_test

import (
	"context"
	"testing"

	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresIdempotencyRepository_CheckOrSet_NewKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresIdempotencyRepository(db)
	ctx := context.Background()
	key := "IDEMP-KEY-001"

	mock.ExpectExec("INSERT INTO idempotency_keys").
		WithArgs(key, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	isNew, err := repo.CheckOrSet(ctx, nil, key)
	require.NoError(t, err)
	assert.True(t, isNew)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresIdempotencyRepository_CheckOrSet_DuplicateKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresIdempotencyRepository(db)
	ctx := context.Background()
	key := "IDEMP-KEY-EXISTS"

	// ON CONFLICT DO NOTHING returns 0 rows affected
	mock.ExpectExec("INSERT INTO idempotency_keys").
		WithArgs(key, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	isNew, err := repo.CheckOrSet(ctx, nil, key)
	require.NoError(t, err)
	assert.False(t, isNew)

	require.NoError(t, mock.ExpectationsWereMet())
}
