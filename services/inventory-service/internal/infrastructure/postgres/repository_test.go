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

func TestPostgresInventoryRepository_GetItemsBySKUs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()

	// Empty SKUs list returns nil without querying DB
	items, err := repo.GetItemsBySKUs(ctx, []string{})
	require.NoError(t, err)
	assert.Nil(t, items)

	// Normal query with multiple SKUs
	skus := []string{"SKU-001", "SKU-002"}
	rows := sqlmock.NewRows([]string{"sku", "warehouse_id", "physical_qty", "reserved_qty", "status", "updated_at"}).
		AddRow("SKU-001", entity.DefaultWarehouseID, 100, 20, "ACTIVE", time.Now().UTC()).
		AddRow("SKU-002", entity.DefaultWarehouseID, 50, 5, "ACTIVE", time.Now().UTC())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at FROM inventory_items WHERE sku IN ($1,$2)")).
		WithArgs("SKU-001", "SKU-002").
		WillReturnRows(rows)

	results, err := repo.GetItemsBySKUs(ctx, skus)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "SKU-001", results[0].SKU)
	assert.Equal(t, 80, results[0].AvailableQty())
	assert.Equal(t, "SKU-002", results[1].SKU)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetBatchesBySKU(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	sku := "SKU-ME-XUNG-GION"

	now := time.Now().UTC()
	b1ID := uuid.New()
	b2ID := uuid.New()
	supplierID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "batch_code", "sku", "supplier_id", "mfg_date", "exp_date", "physical_qty", "reserved_qty", "status", "created_at"}).
		AddRow(b1ID, "LOT-01", sku, supplierID, now.AddDate(0, -1, 0), now.AddDate(0, 2, 0), 50, 10, "ACTIVE", now.AddDate(0, -1, 0)).
		AddRow(b2ID, "LOT-02", sku, supplierID, now.AddDate(0, -2, 0), now.AddDate(0, 4, 0), 100, 0, "ACTIVE", now.AddDate(0, -2, 0))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at FROM batches WHERE sku = $1 ORDER BY exp_date ASC, created_at ASC")).
		WithArgs(sku).
		WillReturnRows(rows)

	batches, err := repo.GetBatchesBySKU(ctx, sku)
	require.NoError(t, err)
	require.Len(t, batches, 2)
	assert.Equal(t, "LOT-01", batches[0].BatchCode)
	assert.Equal(t, 40, batches[0].AvailableQty())
	assert.Equal(t, "LOT-02", batches[1].BatchCode)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetReservationByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()

	resID := uuid.New()
	orderID := "ORDER-RES-BY-ID"
	now := time.Now().UTC()
	allocID := uuid.New()
	batchID := uuid.New()

	rowsRes := sqlmock.NewRows([]string{"id", "order_id", "status", "expires_at", "created_at", "updated_at"}).
		AddRow(resID, orderID, "PENDING", now.Add(15*time.Minute), now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, order_id, status, expires_at, created_at, updated_at FROM stock_reservations WHERE id = $1")).
		WithArgs(resID).
		WillReturnRows(rowsRes)

	rowsAlloc := sqlmock.NewRows([]string{"id", "reservation_id", "sku", "batch_id", "allocated_qty"}).
		AddRow(allocID, resID, "SKU-001", batchID, 10)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, reservation_id, sku, batch_id, allocated_qty FROM stock_reservation_allocations WHERE reservation_id = $1")).
		WithArgs(resID).
		WillReturnRows(rowsAlloc)

	res, err := repo.GetReservationByID(ctx, resID)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, resID, res.ID)
	assert.Equal(t, orderID, res.OrderID)
	require.Len(t, res.Allocations, 1)
	assert.Equal(t, 10, res.Allocations[0].AllocatedQty)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetReservationByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	resID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, order_id, status, expires_at, created_at, updated_at FROM stock_reservations WHERE id = $1")).
		WithArgs(resID).
		WillReturnError(sql.ErrNoRows)

	res, err := repo.GetReservationByID(ctx, resID)
	require.ErrorIs(t, err, entity.ErrReservationNotFound)
	assert.Nil(t, res)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetExpiredPendingReservations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	resID := uuid.New()
	allocID := uuid.New()
	batchID := uuid.New()

	rowsRes := sqlmock.NewRows([]string{"id", "order_id", "status", "expires_at", "created_at", "updated_at"}).
		AddRow(resID, "ORD-EXP-1", "PENDING", now.Add(-5*time.Minute), now.Add(-20*time.Minute), now.Add(-20*time.Minute))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, order_id, status, expires_at, created_at, updated_at FROM stock_reservations WHERE status = 'PENDING' AND expires_at < $1 ORDER BY expires_at ASC LIMIT $2")).
		WithArgs(now, 10).
		WillReturnRows(rowsRes)

	rowsAlloc := sqlmock.NewRows([]string{"id", "reservation_id", "sku", "batch_id", "allocated_qty"}).
		AddRow(allocID, resID, "SKU-001", batchID, 5)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, reservation_id, sku, batch_id, allocated_qty FROM stock_reservation_allocations WHERE reservation_id = $1")).
		WithArgs(resID).
		WillReturnRows(rowsAlloc)

	resList, err := repo.GetExpiredPendingReservations(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, resList, 1)
	assert.Equal(t, resID, resList[0].ID)
	assert.Equal(t, "ORD-EXP-1", resList[0].OrderID)
	require.Len(t, resList[0].Allocations, 1)
	assert.Equal(t, 5, resList[0].Allocations[0].AllocatedQty)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInventoryRepository_GetBatchesNearExpiry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresInventoryRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	threshold := now.Add(45 * 24 * time.Hour)
	batchID := uuid.New()
	supplierID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "batch_code", "sku", "supplier_id", "mfg_date", "exp_date",
		"physical_qty", "reserved_qty", "status", "created_at",
	}).AddRow(batchID, "LOT-001", "SKU-001", supplierID, now.Add(-30*24*time.Hour), now.Add(10*24*time.Hour), 100, 20, "ACTIVE", now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at FROM batches WHERE status IN ('ACTIVE', 'NEAR_EXPIRY') AND exp_date <= $1 ORDER BY exp_date ASC LIMIT $2")).
		WithArgs(threshold, 20).
		WillReturnRows(rows)

	batches, err := repo.GetBatchesNearExpiry(ctx, threshold, 20)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	assert.Equal(t, batchID, batches[0].ID)
	assert.Equal(t, "LOT-001", batches[0].BatchCode)
	assert.Equal(t, entity.BatchStatusActive, batches[0].Status)

	require.NoError(t, mock.ExpectationsWereMet())
}



