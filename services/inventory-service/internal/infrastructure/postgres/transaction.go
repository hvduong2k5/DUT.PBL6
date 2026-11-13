package postgres

import (
	"context"
	"database/sql"
	"errors"

	"dut-pbl6/inventory-service/internal/application/port"
)

// SQLTransaction wraps *sql.Tx to implement port.Transaction.
type SQLTransaction struct {
	Tx *sql.Tx
}

func (t *SQLTransaction) Commit() error {
	if t.Tx == nil {
		return errors.New("nil transaction")
	}
	return t.Tx.Commit()
}

func (t *SQLTransaction) Rollback() error {
	if t.Tx == nil {
		return errors.New("nil transaction")
	}
	return t.Tx.Rollback()
}

// PostgresTxManager implements port.TransactionManager using database/sql.
type PostgresTxManager struct {
	db *sql.DB
}

// NewPostgresTxManager creates a new transaction manager.
func NewPostgresTxManager(db *sql.DB) *PostgresTxManager {
	return &PostgresTxManager{db: db}
}

// ExecuteInTransaction executes the given callback in a database transaction.
// If fn returns an error, the transaction is rolled back; otherwise, it is committed.
func (tm *PostgresTxManager) ExecuteInTransaction(ctx context.Context, fn func(tx port.Transaction) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if tm == nil || tm.db == nil {
		return errors.New("nil db")
	}
	if fn == nil {
		return errors.New("nil transaction function")
	}
	tx, err := tm.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return err
	}

	sqlTx := &SQLTransaction{Tx: tx}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(sqlTx); err != nil {
		return err
	}

	return tx.Commit()
}
