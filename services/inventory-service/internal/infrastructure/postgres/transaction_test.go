package postgres_test

import (
	"context"
	"errors"
	"testing"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type DummyTx struct {
	committed  bool
	rolledBack bool
}

func (d *DummyTx) Commit() error {
	d.committed = true
	return nil
}

func (d *DummyTx) Rollback() error {
	d.rolledBack = true
	return nil
}

func TestSQLTransaction_NilTxHandling(t *testing.T) {
	tx := &postgres.SQLTransaction{Tx: nil}
	assert.Error(t, tx.Commit())
	assert.Error(t, tx.Rollback())
}

func TestTransaction_MockExecution(t *testing.T) {
	dummy := &DummyTx{}
	err := func(tx port.Transaction) error {
		return errors.New("simulated error")
	}(dummy)

	assert.Error(t, err)
	if err != nil {
		_ = dummy.Rollback()
	}
	assert.True(t, dummy.rolledBack)
	assert.False(t, dummy.committed)
}

func TestPostgresTxManager_CommitOnSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	txManager := postgres.NewPostgresTxManager(db)
	err = txManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
		return nil
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTxManager_RollbackOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	txManager := postgres.NewPostgresTxManager(db)
	simulatedErr := errors.New("business logic error")
	err = txManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
		return simulatedErr
	})
	assert.ErrorIs(t, err, simulatedErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTxManager_RollbackOnPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	txManager := postgres.NewPostgresTxManager(db)
	assert.Panics(t, func() {
		_ = txManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
			panic("simulated panic inside tx")
		})
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTxManager_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("network error on commit"))

	txManager := postgres.NewPostgresTxManager(db)
	err = txManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
		return nil
	})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTxManager_NilDB(t *testing.T) {
	txManager := postgres.NewPostgresTxManager(nil)
	err := txManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "nil db", err.Error())

	var nilManager *postgres.PostgresTxManager
	err = nilManager.ExecuteInTransaction(context.Background(), func(tx port.Transaction) error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "nil db", err.Error())
}

func TestPostgresTxManager_NilFn(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	txManager := postgres.NewPostgresTxManager(db)
	err = txManager.ExecuteInTransaction(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, "nil transaction function", err.Error())
}


