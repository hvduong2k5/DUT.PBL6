package postgres_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresInventoryRepository_GetItemBySKU(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	sku := "SKU-001"
	warehouseID := entity.DefaultWarehouseID
	now := time.Now()

	rows := sqlmock.NewRows([]string{"sku", "warehouse_id", "physical_qty", "reserved_qty", "status", "updated_at"}).
		AddRow(sku, warehouseID, 100, 20, "ACTIVE", now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at FROM inventory_items WHERE sku = $1")).
		WithArgs(sku).
		WillReturnRows(rows)

	item, err := repo.GetItemBySKU(ctx, sku)
	require.NoError(t, err)
	assert.Equal(t, sku, item.SKU)
	assert.Equal(t, 100, item.PhysicalQty)
	assert.Equal(t, 20, item.ReservedQty)
	assert.Equal(t, 80, item.AvailableQty())
	assert.Equal(t, entity.ItemStatusActive, item.Status)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetItemBySKU_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	sku := "SKU-NOT-EXIST"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at FROM inventory_items WHERE sku = $1")).
		WithArgs(sku).
		WillReturnError(sql.ErrNoRows)

	item, err := repo.GetItemBySKU(ctx, sku)
	assert.ErrorIs(t, err, entity.ErrItemNotFound)
	assert.Nil(t, item)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetItemBySKUForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	sku := "SKU-FOR-UPDATE"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"sku", "warehouse_id", "physical_qty", "reserved_qty", "status", "updated_at"}).
		AddRow(sku, entity.DefaultWarehouseID, 50, 10, "ACTIVE", now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at FROM inventory_items WHERE sku = $1 FOR UPDATE")).
		WithArgs(sku).
		WillReturnRows(rows)

	item, err := repo.GetItemBySKUForUpdate(ctx, nil, sku)
	require.NoError(t, err)
	assert.Equal(t, sku, item.SKU)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_UpdateItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	item, _ := entity.NewInventoryItem("SKU-UPD", entity.DefaultWarehouseID, 100)
	item.ReservedQty = 30

	mock.ExpectExec("UPDATE inventory_items").
		WithArgs(100, 30, "ACTIVE", sqlmock.AnyArg(), "SKU-UPD").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateItem(ctx, nil, item)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetActiveBatchesBySKUForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	sku := "SKU-001"
	batchID := uuid.New()
	suppID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "batch_code", "sku", "supplier_id", "mfg_date", "exp_date", "physical_qty", "reserved_qty", "status", "created_at",
	}).AddRow(batchID, "LOT-1", sku, suppID, now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour), 100, 0, "ACTIVE", now)

	mock.ExpectQuery("SELECT (.+) FROM batches WHERE sku = \\$1 AND status IN \\('ACTIVE', 'NEAR_EXPIRY'\\)").
		WithArgs(sku).
		WillReturnRows(rows)

	batches, err := repo.GetActiveBatchesBySKUForUpdate(ctx, nil, sku)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	assert.Equal(t, batchID, batches[0].ID)
	assert.Equal(t, entity.BatchStatusActive, batches[0].Status)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetBatchByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	batchID := uuid.New()
	suppID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "batch_code", "sku", "supplier_id", "mfg_date", "exp_date", "physical_qty", "reserved_qty", "status", "created_at",
	}).AddRow(batchID, "LOT-1", "SKU-1", suppID, now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour), 100, 10, "ACTIVE", now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at FROM batches WHERE id = $1")).
		WithArgs(batchID).
		WillReturnRows(rows)

	batch, err := repo.GetBatchByID(ctx, nil, batchID)
	require.NoError(t, err)
	assert.Equal(t, batchID, batch.ID)
	assert.Equal(t, 90, batch.AvailableQty())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetBatchesByIDsForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	batchID1 := uuid.New()
	batchID2 := uuid.New()
	suppID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "batch_code", "sku", "supplier_id", "mfg_date", "exp_date", "physical_qty", "reserved_qty", "status", "created_at",
	}).AddRow(batchID1, "LOT-1", "SKU-1", suppID, now, now.Add(30*24*time.Hour), 50, 0, "ACTIVE", now).
		AddRow(batchID2, "LOT-2", "SKU-1", suppID, now, now.Add(60*24*time.Hour), 50, 0, "ACTIVE", now)

	mock.ExpectQuery("SELECT (.+) FROM batches WHERE id IN \\(\\$1,\\$2\\) FOR UPDATE").
		WithArgs(batchID1, batchID2).
		WillReturnRows(rows)

	batches, err := repo.GetBatchesByIDsForUpdate(ctx, nil, []uuid.UUID{batchID1, batchID2})
	require.NoError(t, err)
	require.Len(t, batches, 2)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_CreateAndGetReservation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	orderID := "ORD-TEST-001"
	batchID := uuid.New()

	alloc := entity.ReservationItemAllocation{
		ID:           uuid.New(),
		SKU:          "SKU-001",
		BatchID:      batchID,
		AllocatedQty: 10,
	}
	res, err := entity.NewStockReservation(orderID, entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)

	// Expect Insert reservation
	mock.ExpectExec("INSERT INTO stock_reservations").
		WithArgs(res.ID, res.OrderID, string(res.Status), res.ExpiresAt, res.CreatedAt, res.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect Insert allocation
	mock.ExpectExec("INSERT INTO stock_reservation_allocations").
		WithArgs(res.Allocations[0].ID, res.ID, alloc.SKU, alloc.BatchID, alloc.AllocatedQty).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateReservation(ctx, nil, res)
	require.NoError(t, err)

	// Expect GetReservationByOrderID
	resRows := sqlmock.NewRows([]string{"id", "order_id", "status", "expires_at", "created_at", "updated_at"}).
		AddRow(res.ID, orderID, "PENDING", res.ExpiresAt, res.CreatedAt, res.UpdatedAt)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, order_id, status, expires_at, created_at, updated_at FROM stock_reservations WHERE order_id = $1")).
		WithArgs(orderID).
		WillReturnRows(resRows)

	allocRows := sqlmock.NewRows([]string{"id", "reservation_id", "sku", "batch_id", "allocated_qty"}).
		AddRow(alloc.ID, res.ID, alloc.SKU, alloc.BatchID, alloc.AllocatedQty)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, reservation_id, sku, batch_id, allocated_qty FROM stock_reservation_allocations WHERE reservation_id = $1")).
		WithArgs(res.ID).
		WillReturnRows(allocRows)

	fetched, err := repo.GetReservationByOrderID(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, orderID, fetched.OrderID)
	require.Len(t, fetched.Allocations, 1)
	assert.Equal(t, 10, fetched.Allocations[0].AllocatedQty)

	// Expect UpdateReservation
	mock.ExpectExec("UPDATE stock_reservations").
		WithArgs("RELEASED", sqlmock.AnyArg(), res.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	fetched.Status = entity.ReservationStatusReleased
	err = repo.UpdateReservation(ctx, nil, fetched)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
