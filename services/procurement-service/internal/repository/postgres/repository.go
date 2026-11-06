package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) CreateSupplier(ctx context.Context, supplier domain.Supplier) (domain.Supplier, error) {
	const query = `INSERT INTO suppliers (id, code, name, supplier_type, contact_name, phone, email, address, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, supplier.ID, supplier.Code, supplier.Name, supplier.SupplierType,
		supplier.ContactName, supplier.Phone, supplier.Email, supplier.Address, supplier.Status).
		Scan(&supplier.CreatedAt, &supplier.UpdatedAt)
	if err != nil {
		return domain.Supplier{}, mapError(err)
	}
	return supplier, nil
}

func (r *Repository) UpdateSupplier(ctx context.Context, id uuid.UUID, supplier domain.Supplier) (domain.Supplier, error) {
	const query = `UPDATE suppliers SET code=$2, name=$3, supplier_type=$4, contact_name=$5, phone=$6,
		email=$7, address=$8, updated_at=now()
		WHERE id=$1 RETURNING id, code, name, supplier_type, contact_name, phone, email, address, status, created_at, updated_at`
	item, err := scanSupplier(r.pool.QueryRow(ctx, query, id, supplier.Code, supplier.Name, supplier.SupplierType,
		supplier.ContactName, supplier.Phone, supplier.Email, supplier.Address))
	if err != nil {
		return domain.Supplier{}, mapError(err)
	}
	return item, nil
}

func (r *Repository) GetSupplier(ctx context.Context, id uuid.UUID) (domain.Supplier, error) {
	return scanSupplier(r.pool.QueryRow(ctx, supplierSelect+` WHERE id=$1`, id))
}

func (r *Repository) ListSuppliers(ctx context.Context, limit, offset int) ([]domain.Supplier, error) {
	rows, err := r.pool.Query(ctx, supplierSelect+` ORDER BY name, id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Supplier, 0)
	for rows.Next() {
		item, scanErr := scanSupplierRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SetSupplierStatus(ctx context.Context, id uuid.UUID, status domain.SupplierStatus) (domain.Supplier, error) {
	const query = `UPDATE suppliers SET status=$2, updated_at=now() WHERE id=$1 RETURNING id, code, name, supplier_type, contact_name, phone, email, address, status, created_at, updated_at`
	return scanSupplier(r.pool.QueryRow(ctx, query, id, status))
}

func (r *Repository) CreateMaterial(ctx context.Context, material domain.RawMaterial) (domain.RawMaterial, error) {
	const query = `INSERT INTO raw_materials (id, code, name, base_unit, description, status)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, material.ID, material.Code, material.Name, material.BaseUnit, material.Description, material.Status).
		Scan(&material.CreatedAt, &material.UpdatedAt)
	if err != nil {
		return domain.RawMaterial{}, mapError(err)
	}
	return material, nil
}

func (r *Repository) SetMaterialStatus(ctx context.Context, id uuid.UUID, status domain.MaterialStatus) (domain.RawMaterial, error) {
	const query = `UPDATE raw_materials SET status=$2, updated_at=now() WHERE id=$1 RETURNING id, code, name, base_unit, description, status, created_at, updated_at`
	return scanMaterial(r.pool.QueryRow(ctx, query, id, status))
}

func (r *Repository) GetMaterial(ctx context.Context, id uuid.UUID) (domain.RawMaterial, error) {
	return scanMaterial(r.pool.QueryRow(ctx, materialSelect+` WHERE id=$1`, id))
}

func (r *Repository) ListMaterials(ctx context.Context, limit, offset int) ([]domain.RawMaterial, error) {
	rows, err := r.pool.Query(ctx, materialSelect+` ORDER BY name, id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.RawMaterial, 0)
	for rows.Next() {
		item, scanErr := scanMaterialRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateOffer(ctx context.Context, offer domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error) {
	const query = `INSERT INTO supplier_material_offers
		(id, supplier_id, material_id, supplier_sku, unit, unit_price_minor, currency, minimum_quantity, lead_time_days, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, offer.ID, offer.SupplierID, offer.MaterialID, offer.SupplierSKU,
		offer.Unit, offer.UnitPriceMinor, offer.Currency, offer.MinimumQty, offer.LeadTimeDays, offer.Active).
		Scan(&offer.CreatedAt, &offer.UpdatedAt)
	if err != nil {
		return domain.SupplierMaterialOffer{}, mapError(err)
	}
	return offer, nil
}

func (r *Repository) ListSupplierOffers(ctx context.Context, supplierID uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	const query = `SELECT id, supplier_id, material_id, supplier_sku, unit, unit_price_minor,
		currency, minimum_quantity::text, lead_time_days, active, created_at, updated_at
		FROM supplier_material_offers WHERE supplier_id=$1 ORDER BY material_id, id`
	rows, err := r.pool.Query(ctx, query, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.SupplierMaterialOffer, 0)
	for rows.Next() {
		var item domain.SupplierMaterialOffer
		if err := rows.Scan(&item.ID, &item.SupplierID, &item.MaterialID, &item.SupplierSKU, &item.Unit,
			&item.UnitPriceMinor, &item.Currency, &item.MinimumQty, &item.LeadTimeDays, &item.Active,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreatePurchaseOrder(ctx context.Context, order domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const query = `INSERT INTO purchase_orders
		(id, po_number, supplier_id, supplier_name, status, currency, total_amount_minor, notes,
		expected_delivery_date, created_by, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, query, order.ID, order.Number, order.SupplierID, order.SupplierName, order.Status,
		order.Currency, order.TotalAmountMinor, order.Notes, order.ExpectedDeliveryDate, order.CreatedBy, order.Version).
		Scan(&order.CreatedAt, &order.UpdatedAt); err != nil {
		return domain.PurchaseOrder{}, mapError(err)
	}
	if err := insertPurchaseOrderLines(ctx, tx, order); err != nil {
		return domain.PurchaseOrder{}, mapError(err)
	}
	if err := insertPurchaseOrderHistory(ctx, tx, order.ID, "", string(order.Status), "CREATED", order.CreatedBy, ""); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return order, nil
}

func (r *Repository) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (domain.PurchaseOrder, error) {
	order, err := scanPurchaseOrder(r.pool.QueryRow(ctx, purchaseOrderSelect+` WHERE id=$1`, id))
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	rows, err := r.pool.Query(ctx, purchaseOrderLineSelect+` WHERE purchase_order_id=$1 ORDER BY line_number`, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer rows.Close()
	order.Lines = make([]domain.PurchaseOrderLine, 0)
	for rows.Next() {
		var line domain.PurchaseOrderLine
		if err := rows.Scan(&line.ID, &line.LineNumber, &line.MaterialID, &line.MaterialCode, &line.MaterialName,
			&line.Quantity, &line.ReceivedQuantity, &line.Unit, &line.UnitPriceMinor, &line.LineAmountMinor); err != nil {
			return domain.PurchaseOrder{}, err
		}
		order.Lines = append(order.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return order, nil
}

func (r *Repository) ListPurchaseOrders(ctx context.Context, supplierID uuid.UUID, status domain.PurchaseOrderStatus, limit, offset int) ([]domain.PurchaseOrder, error) {
	var supplierFilter any
	if supplierID != uuid.Nil {
		supplierFilter = supplierID
	}
	var statusFilter any
	if status != "" {
		statusFilter = string(status)
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM purchase_orders
		WHERE ($1::uuid IS NULL OR supplier_id=$1) AND ($2::text IS NULL OR status=$2)
		ORDER BY created_at DESC, id LIMIT $3 OFFSET $4`, supplierFilter, statusFilter, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return nil, rowsErr
	}
	items := make([]domain.PurchaseOrder, 0, len(ids))
	for _, id := range ids {
		item, err := r.GetPurchaseOrder(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) SavePurchaseOrder(ctx context.Context, order domain.PurchaseOrder, expectedVersion int64, previousStatus domain.PurchaseOrderStatus, action, actor, reason string) (domain.PurchaseOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const query = `UPDATE purchase_orders SET supplier_id=$3, supplier_name=$4, status=$5, currency=$6,
		total_amount_minor=$7, notes=$8, expected_delivery_date=$9, submitted_by=$10, submitted_at=$11,
		approved_by=$12, approved_at=$13, rejected_by=$14, rejected_at=$15, rejection_reason=$16,
		sent_by=$17, sent_at=$18, cancelled_by=$19, cancelled_at=$20, cancellation_reason=$21,
		updated_at=now(), version=version+1
		WHERE id=$1 AND version=$2
		RETURNING created_at, updated_at, version`
	err = tx.QueryRow(ctx, query, order.ID, expectedVersion, order.SupplierID, order.SupplierName, order.Status,
		order.Currency, order.TotalAmountMinor, order.Notes, order.ExpectedDeliveryDate,
		emptyToNil(order.SubmittedBy), order.SubmittedAt, emptyToNil(order.ApprovedBy), order.ApprovedAt,
		emptyToNil(order.RejectedBy), order.RejectedAt, order.RejectionReason,
		emptyToNil(order.SentBy), order.SentAt, emptyToNil(order.CancelledBy), order.CancelledAt, order.CancellationReason).
		Scan(&order.CreatedAt, &order.UpdatedAt, &order.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	if err != nil {
		return domain.PurchaseOrder{}, mapError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM purchase_order_lines WHERE purchase_order_id=$1`, order.ID); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := insertPurchaseOrderLines(ctx, tx, order); err != nil {
		return domain.PurchaseOrder{}, mapError(err)
	}
	if err := insertPurchaseOrderHistory(ctx, tx, order.ID, string(previousStatus), string(order.Status), action, actor, reason); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return order, nil
}

func insertPurchaseOrderLines(ctx context.Context, tx pgx.Tx, order domain.PurchaseOrder) error {
	const query = `INSERT INTO purchase_order_lines
		(id, purchase_order_id, line_number, material_id, material_code, material_name, quantity, received_quantity, unit, unit_price_minor, line_amount_minor)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	for _, line := range order.Lines {
		receivedQuantity := line.ReceivedQuantity
		if receivedQuantity == "" {
			receivedQuantity = "0"
		}
		if _, err := tx.Exec(ctx, query, line.ID, order.ID, line.LineNumber, line.MaterialID, line.MaterialCode,
			line.MaterialName, line.Quantity, receivedQuantity, line.Unit, line.UnitPriceMinor, line.LineAmountMinor); err != nil {
			return err
		}
	}
	return nil
}

func insertPurchaseOrderHistory(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, fromStatus, toStatus, action, actor, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO purchase_order_status_history
		(id, purchase_order_id, from_status, to_status, action, actor_id, reason)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7)`, uuid.New(), orderID, fromStatus, toStatus, action, actor, reason)
	return err
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

const purchaseOrderSelect = `SELECT id, po_number, supplier_id, supplier_name, status, currency,
	total_amount_minor, notes, expected_delivery_date, created_by, submitted_by, submitted_at,
	approved_by, approved_at, rejected_by, rejected_at, rejection_reason, sent_by, sent_at,
	cancelled_by, cancelled_at, cancellation_reason, created_at, updated_at, version FROM purchase_orders`

const purchaseOrderLineSelect = `SELECT id, line_number, material_id, material_code, material_name,
	quantity::text, received_quantity::text, unit, unit_price_minor, line_amount_minor FROM purchase_order_lines`

func scanPurchaseOrder(row rowScanner) (domain.PurchaseOrder, error) {
	var order domain.PurchaseOrder
	var expected pgtype.Date
	var submitted, approved, rejected, sent, cancelled pgtype.Timestamptz
	var submittedBy, approvedBy, rejectedBy, rejectionReason, sentBy, cancelledBy, cancellationReason pgtype.Text
	err := row.Scan(&order.ID, &order.Number, &order.SupplierID, &order.SupplierName, &order.Status, &order.Currency,
		&order.TotalAmountMinor, &order.Notes, &expected, &order.CreatedBy, &submittedBy, &submitted,
		&approvedBy, &approved, &rejectedBy, &rejected, &rejectionReason, &sentBy, &sent,
		&cancelledBy, &cancelled, &cancellationReason, &order.CreatedAt, &order.UpdatedAt, &order.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if expected.Valid {
		value := expected.Time
		order.ExpectedDeliveryDate = &value
	}
	order.SubmittedBy, order.ApprovedBy, order.RejectedBy = submittedBy.String, approvedBy.String, rejectedBy.String
	order.RejectionReason, order.SentBy, order.CancelledBy = rejectionReason.String, sentBy.String, cancelledBy.String
	order.CancellationReason = cancellationReason.String
	order.SubmittedAt = timestampPointer(submitted)
	order.ApprovedAt = timestampPointer(approved)
	order.RejectedAt = timestampPointer(rejected)
	order.SentAt = timestampPointer(sent)
	order.CancelledAt = timestampPointer(cancelled)
	return order, nil
}

func timestampPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

const supplierSelect = `SELECT id, code, name, supplier_type, contact_name, phone, email, address, status, created_at, updated_at FROM suppliers`
const materialSelect = `SELECT id, code, name, base_unit, description, status, created_at, updated_at FROM raw_materials`

type rowScanner interface{ Scan(...any) error }

func scanSupplier(row rowScanner) (domain.Supplier, error) {
	var item domain.Supplier
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.SupplierType, &item.ContactName, &item.Phone,
		&item.Email, &item.Address, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Supplier{}, domain.ErrNotFound
	}
	return item, err
}

func scanSupplierRow(row rowScanner) (domain.Supplier, error) { return scanSupplier(row) }

func scanMaterial(row rowScanner) (domain.RawMaterial, error) {
	var item domain.RawMaterial
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.BaseUnit, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	return item, err
}

func scanMaterialRow(row rowScanner) (domain.RawMaterial, error) { return scanMaterial(row) }

func mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrConflict, strings.TrimSpace(pgErr.ConstraintName))
		case "23503":
			return fmt.Errorf("%w: referenced supplier or material does not exist", domain.ErrInvalidArgument)
		case "23514", "22P02":
			return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, pgErr.Message)
		}
	}
	return err
}
