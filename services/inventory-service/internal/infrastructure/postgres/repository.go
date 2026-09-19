package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
)

type queryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// PostgresInventoryRepository implements port.InventoryRepository.
type PostgresInventoryRepository struct {
	db *sql.DB
}

// NewPostgresInventoryRepository creates a new PostgresInventoryRepository.
func NewPostgresInventoryRepository(db *sql.DB) *PostgresInventoryRepository {
	return &PostgresInventoryRepository{db: db}
}

func (r *PostgresInventoryRepository) getExecutor(tx port.Transaction) queryExecutor {
	if tx != nil {
		if sqlTx, ok := tx.(*SQLTransaction); ok && sqlTx.Tx != nil {
			return sqlTx.Tx
		}
	}
	return r.db
}

// GetItemBySKU retrieves an inventory item by SKU.
func (r *PostgresInventoryRepository) GetItemBySKU(ctx context.Context, sku string) (*entity.InventoryItem, error) {
	query := `SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at
              FROM inventory_items WHERE sku = $1`
	return r.scanItem(r.db.QueryRowContext(ctx, query, sku))
}

// GetItemBySKUForUpdate retrieves an inventory item locking the row with SELECT ... FOR UPDATE.
func (r *PostgresInventoryRepository) GetItemBySKUForUpdate(ctx context.Context, tx port.Transaction, sku string) (*entity.InventoryItem, error) {
	query := `SELECT sku, warehouse_id, physical_qty, reserved_qty, status, updated_at
              FROM inventory_items WHERE sku = $1 FOR UPDATE`
	return r.scanItem(r.getExecutor(tx).QueryRowContext(ctx, query, sku))
}

func (r *PostgresInventoryRepository) scanItem(row *sql.Row) (*entity.InventoryItem, error) {
	item := &entity.InventoryItem{}
	var status string
	err := row.Scan(&item.SKU, &item.WarehouseID, &item.PhysicalQty, &item.ReservedQty, &status, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrItemNotFound
		}
		return nil, err
	}
	item.Status = entity.ItemStatus(status)
	return item, nil
}

// UpdateItem updates physical and reserved quantities for an inventory item.
func (r *PostgresInventoryRepository) UpdateItem(ctx context.Context, tx port.Transaction, item *entity.InventoryItem) error {
	query := `UPDATE inventory_items 
              SET physical_qty = $1, reserved_qty = $2, status = $3, updated_at = $4 
              WHERE sku = $5`
	_, err := r.getExecutor(tx).ExecContext(ctx, query,
		item.PhysicalQty, item.ReservedQty, string(item.Status), time.Now().UTC(), item.SKU)
	return err
}

// GetActiveBatchesBySKUForUpdate queries active and near-expiry batches for a SKU ordered by FEFO and locks them.
func (r *PostgresInventoryRepository) GetActiveBatchesBySKUForUpdate(ctx context.Context, tx port.Transaction, sku string) ([]*entity.Batch, error) {
	query := `SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at
              FROM batches
              WHERE sku = $1 AND status IN ('ACTIVE', 'NEAR_EXPIRY')
              ORDER BY exp_date ASC, created_at ASC
              FOR UPDATE`
	rows, err := r.getExecutor(tx).QueryContext(ctx, query, sku)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []*entity.Batch
	for rows.Next() {
		b := &entity.Batch{}
		var status string
		err := rows.Scan(
			&b.ID, &b.BatchCode, &b.SKU, &b.SupplierID,
			&b.MfgDate, &b.ExpDate, &b.PhysicalQty, &b.ReservedQty,
			&status, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		b.Status = entity.BatchStatus(status)
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// GetBatchByID retrieves a single batch by ID (locked for update if in transaction).
func (r *PostgresInventoryRepository) GetBatchByID(ctx context.Context, tx port.Transaction, batchID uuid.UUID) (*entity.Batch, error) {
	query := `SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at
              FROM batches WHERE id = $1`
	if tx != nil {
		query += " FOR UPDATE"
	}
	row := r.getExecutor(tx).QueryRowContext(ctx, query, batchID)

	b := &entity.Batch{}
	var status string
	err := row.Scan(
		&b.ID, &b.BatchCode, &b.SKU, &b.SupplierID,
		&b.MfgDate, &b.ExpDate, &b.PhysicalQty, &b.ReservedQty,
		&status, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrInvalidBatchData
		}
		return nil, err
	}
	b.Status = entity.BatchStatus(status)
	return b, nil
}

// GetBatchesByIDsForUpdate retrieves and locks multiple batches by their IDs.
func (r *PostgresInventoryRepository) GetBatchesByIDsForUpdate(ctx context.Context, tx port.Transaction, batchIDs []uuid.UUID) ([]*entity.Batch, error) {
	if len(batchIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(batchIDs))
	args := make([]any, len(batchIDs))
	for idx, id := range batchIDs {
		placeholders[idx] = fmt.Sprintf("$%d", idx+1)
		args[idx] = id
	}

	query := fmt.Sprintf(`SELECT id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at
              FROM batches
              WHERE id IN (%s)
              FOR UPDATE`, strings.Join(placeholders, ","))

	rows, err := r.getExecutor(tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []*entity.Batch
	for rows.Next() {
		b := &entity.Batch{}
		var status string
		err := rows.Scan(
			&b.ID, &b.BatchCode, &b.SKU, &b.SupplierID,
			&b.MfgDate, &b.ExpDate, &b.PhysicalQty, &b.ReservedQty,
			&status, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		b.Status = entity.BatchStatus(status)
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// UpdateBatch updates quantities and status for a batch.
func (r *PostgresInventoryRepository) UpdateBatch(ctx context.Context, tx port.Transaction, batch *entity.Batch) error {
	query := `UPDATE batches 
              SET physical_qty = $1, reserved_qty = $2, status = $3 
              WHERE id = $4`
	_, err := r.getExecutor(tx).ExecContext(ctx, query,
		batch.PhysicalQty, batch.ReservedQty, string(batch.Status), batch.ID)
	return err
}

// CreateReservation persists a stock reservation and its allocations.
func (r *PostgresInventoryRepository) CreateReservation(ctx context.Context, tx port.Transaction, res *entity.StockReservation) error {
	queryRes := `INSERT INTO stock_reservations (id, order_id, status, expires_at, created_at, updated_at)
                 VALUES ($1, $2, $3, $4, $5, $6)`
	exec := r.getExecutor(tx)
	_, err := exec.ExecContext(ctx, queryRes,
		res.ID, res.OrderID, string(res.Status), res.ExpiresAt, res.CreatedAt, res.UpdatedAt)
	if err != nil {
		return err
	}

	queryAlloc := `INSERT INTO stock_reservation_allocations (id, reservation_id, sku, batch_id, allocated_qty)
                   VALUES ($1, $2, $3, $4, $5)`
	for _, alloc := range res.Allocations {
		_, err := exec.ExecContext(ctx, queryAlloc,
			alloc.ID, res.ID, alloc.SKU, alloc.BatchID, alloc.AllocatedQty)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetReservationByOrderID retrieves a reservation and its allocations by orderID.
func (r *PostgresInventoryRepository) GetReservationByOrderID(ctx context.Context, orderID string) (*entity.StockReservation, error) {
	query := `SELECT id, order_id, status, expires_at, created_at, updated_at
              FROM stock_reservations WHERE order_id = $1`
	return r.scanReservation(ctx, r.db.QueryRowContext(ctx, query, orderID), nil)
}

// GetReservationByOrderIDForUpdate retrieves a reservation locking the row with FOR UPDATE.
func (r *PostgresInventoryRepository) GetReservationByOrderIDForUpdate(ctx context.Context, tx port.Transaction, orderID string) (*entity.StockReservation, error) {
	query := `SELECT id, order_id, status, expires_at, created_at, updated_at
              FROM stock_reservations WHERE order_id = $1 FOR UPDATE`
	return r.scanReservation(ctx, r.getExecutor(tx).QueryRowContext(ctx, query, orderID), tx)
}

func (r *PostgresInventoryRepository) scanReservation(ctx context.Context, row *sql.Row, tx port.Transaction) (*entity.StockReservation, error) {
	res := &entity.StockReservation{}
	var status string
	err := row.Scan(&res.ID, &res.OrderID, &status, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrReservationNotFound
		}
		return nil, err
	}
	res.Status = entity.ReservationStatus(status)

	// Fetch allocations
	queryAlloc := `SELECT id, reservation_id, sku, batch_id, allocated_qty
                   FROM stock_reservation_allocations WHERE reservation_id = $1`
	rows, err := r.getExecutor(tx).QueryContext(ctx, queryAlloc, res.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var alloc entity.ReservationItemAllocation
		if err := rows.Scan(&alloc.ID, &alloc.ReservationID, &alloc.SKU, &alloc.BatchID, &alloc.AllocatedQty); err != nil {
			return nil, err
		}
		res.Allocations = append(res.Allocations, alloc)
	}

	return res, rows.Err()
}

// UpdateReservation updates status and timestamp of a reservation.
func (r *PostgresInventoryRepository) UpdateReservation(ctx context.Context, tx port.Transaction, res *entity.StockReservation) error {
	query := `UPDATE stock_reservations 
              SET status = $1, updated_at = $2 
              WHERE id = $3`
	_, err := r.getExecutor(tx).ExecContext(ctx, query,
		string(res.Status), time.Now().UTC(), res.ID)
	return err
}
