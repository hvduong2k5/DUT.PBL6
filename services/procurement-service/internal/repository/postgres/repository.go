package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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
	const query = `INSERT INTO suppliers (id, code, name, supplier_type, contact_name, phone, email, address, legal_identifier, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, supplier.ID, supplier.Code, supplier.Name, supplier.SupplierType,
		supplier.ContactName, supplier.Phone, supplier.Email, supplier.Address, supplier.LegalIdentifier, supplier.Status).
		Scan(&supplier.CreatedAt, &supplier.UpdatedAt)
	if err != nil {
		return domain.Supplier{}, mapError(err)
	}
	return supplier, nil
}

func (r *Repository) UpdateSupplier(ctx context.Context, id uuid.UUID, supplier domain.Supplier) (domain.Supplier, error) {
	const query = `UPDATE suppliers SET code=$2, name=$3, supplier_type=$4, contact_name=$5, phone=$6,
		email=$7, address=$8, legal_identifier=$9, updated_at=now()
		WHERE id=$1 RETURNING id, code, name, supplier_type, contact_name, phone, email, address, legal_identifier, status, created_at, updated_at`
	item, err := scanSupplier(r.pool.QueryRow(ctx, query, id, supplier.Code, supplier.Name, supplier.SupplierType,
		supplier.ContactName, supplier.Phone, supplier.Email, supplier.Address, supplier.LegalIdentifier))
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
	const query = `UPDATE suppliers SET status=$2, updated_at=now() WHERE id=$1 RETURNING id, code, name, supplier_type, contact_name, phone, email, address, legal_identifier, status, created_at, updated_at`
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Supplier{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanSupplier(tx.QueryRow(ctx, query, id, status))
	if err != nil {
		return domain.Supplier{}, err
	}
	if status == domain.SupplierSuspended {
		if _, err := tx.Exec(ctx, `UPDATE raw_materials SET preferred_supplier_id=NULL,updated_at=now() WHERE preferred_supplier_id=$1`, id); err != nil {
			return domain.Supplier{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Supplier{}, err
	}
	return item, nil
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

func (r *Repository) UpdateMaterial(ctx context.Context, id uuid.UUID, material domain.RawMaterial) (domain.RawMaterial, error) {
	return scanMaterial(r.pool.QueryRow(ctx, `UPDATE raw_materials SET code=$2,name=$3,base_unit=$4,description=$5,updated_at=now() WHERE id=$1 RETURNING id,code,name,base_unit,description,status,preferred_supplier_id,created_at,updated_at`, id, material.Code, material.Name, material.BaseUnit, material.Description))
}

func (r *Repository) SetPreferredSupplier(ctx context.Context, materialID, supplierID uuid.UUID) (domain.RawMaterial, error) {
	var preferred any
	if supplierID != uuid.Nil {
		preferred = supplierID
	}
	item, err := scanMaterial(r.pool.QueryRow(ctx, `UPDATE raw_materials m SET preferred_supplier_id=$2,updated_at=now() WHERE m.id=$1 AND ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM suppliers s JOIN supplier_material_offers o ON o.supplier_id=s.id WHERE s.id=$2 AND s.status='ACTIVE' AND o.material_id=m.id AND o.active AND o.effective_from<=now() AND (o.effective_to IS NULL OR o.effective_to>now()))) RETURNING m.id,m.code,m.name,m.base_unit,m.description,m.status,m.preferred_supplier_id,m.created_at,m.updated_at`, materialID, preferred))
	if errors.Is(err, domain.ErrNotFound) && supplierID != uuid.Nil {
		return domain.RawMaterial{}, domain.ErrInvalidState
	}
	return item, err
}

func (r *Repository) SetMaterialStatus(ctx context.Context, id uuid.UUID, status domain.MaterialStatus) (domain.RawMaterial, error) {
	const query = `UPDATE raw_materials SET status=$2, updated_at=now() WHERE id=$1 RETURNING id, code, name, base_unit, description, status, preferred_supplier_id, created_at, updated_at`
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lockedSupplier uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM suppliers WHERE id=$1 AND status='ACTIVE' FOR UPDATE`, offer.SupplierID).Scan(&lockedSupplier); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SupplierMaterialOffer{}, domain.ErrInvalidState
		}
		return domain.SupplierMaterialOffer{}, err
	}
	var lockedMaterial uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM raw_materials WHERE id=$1 AND status='ACTIVE' AND base_unit=$2 FOR UPDATE`, offer.MaterialID, offer.Unit).Scan(&lockedMaterial); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SupplierMaterialOffer{}, domain.ErrInvalidState
		}
		return domain.SupplierMaterialOffer{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE supplier_material_offers SET effective_to=$3,updated_at=now() WHERE supplier_id=$1 AND material_id=$2 AND active AND effective_to IS NULL AND effective_from<$3`, offer.SupplierID, offer.MaterialID, offer.EffectiveFrom); err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	var overlap bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2 AND active AND effective_from<COALESCE($4::timestamptz,'infinity') AND COALESCE(effective_to,'infinity')>$3::timestamptz)`, offer.SupplierID, offer.MaterialID, offer.EffectiveFrom, offer.EffectiveTo).Scan(&overlap); err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	if overlap {
		return domain.SupplierMaterialOffer{}, domain.ErrConflict
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2`, offer.SupplierID, offer.MaterialID).Scan(&offer.Version); err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO supplier_material_offers (id,supplier_id,material_id,supplier_sku,unit,unit_price_minor,currency,minimum_quantity,lead_time_days,active,version,effective_from,effective_to) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING created_at,updated_at`, offer.ID, offer.SupplierID, offer.MaterialID, offer.SupplierSKU, offer.Unit, offer.UnitPriceMinor, offer.Currency, offer.MinimumQty, offer.LeadTimeDays, offer.Active, offer.Version, offer.EffectiveFrom, offer.EffectiveTo).Scan(&offer.CreatedAt, &offer.UpdatedAt)
	if err != nil {
		return domain.SupplierMaterialOffer{}, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	return offer, nil
}

func (r *Repository) SetOfferActive(ctx context.Context, supplierID, offerID uuid.UUID, active bool) (domain.SupplierMaterialOffer, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item domain.SupplierMaterialOffer
	var effectiveTo pgtype.Timestamptz
	err = tx.QueryRow(ctx, `UPDATE supplier_material_offers SET active=$3,updated_at=now() WHERE supplier_id=$1 AND id=$2 RETURNING id,supplier_id,material_id,supplier_sku,unit,unit_price_minor,currency,minimum_quantity::text,lead_time_days,active,version,effective_from,effective_to,created_at,updated_at`, supplierID, offerID, active).
		Scan(&item.ID, &item.SupplierID, &item.MaterialID, &item.SupplierSKU, &item.Unit, &item.UnitPriceMinor, &item.Currency, &item.MinimumQty, &item.LeadTimeDays, &item.Active, &item.Version, &item.EffectiveFrom, &effectiveTo, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SupplierMaterialOffer{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SupplierMaterialOffer{}, mapError(err)
	}
	if effectiveTo.Valid {
		end := effectiveTo.Time
		item.EffectiveTo = &end
	}
	if !active {
		if _, err := tx.Exec(ctx, `UPDATE raw_materials m SET preferred_supplier_id=NULL,updated_at=now() WHERE m.id=$1 AND m.preferred_supplier_id=$2 AND NOT EXISTS(SELECT 1 FROM supplier_material_offers o JOIN suppliers s ON s.id=o.supplier_id WHERE o.material_id=m.id AND o.supplier_id=$2 AND o.active AND s.status='ACTIVE' AND o.effective_from<=now() AND (o.effective_to IS NULL OR o.effective_to>now()))`, item.MaterialID, supplierID); err != nil {
			return domain.SupplierMaterialOffer{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	return item, nil
}

func (r *Repository) ListSupplierOffers(ctx context.Context, supplierID uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return r.listOffers(ctx, `supplier_id`, supplierID)
}

func (r *Repository) ListMaterialOffers(ctx context.Context, materialID uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return r.listOffers(ctx, `material_id`, materialID)
}

func (r *Repository) listOffers(ctx context.Context, column string, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	if column != "supplier_id" && column != "material_id" {
		return nil, domain.ErrInvalidArgument
	}
	query := `SELECT id, supplier_id, material_id, supplier_sku, unit, unit_price_minor,
		currency, minimum_quantity::text, lead_time_days, active, version, effective_from, effective_to, created_at, updated_at
		FROM supplier_material_offers WHERE ` + column + `=$1 ORDER BY supplier_id, material_id, version DESC`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.SupplierMaterialOffer, 0)
	for rows.Next() {
		var item domain.SupplierMaterialOffer
		var effectiveTo pgtype.Timestamptz
		if err := rows.Scan(&item.ID, &item.SupplierID, &item.MaterialID, &item.SupplierSKU, &item.Unit,
			&item.UnitPriceMinor, &item.Currency, &item.MinimumQty, &item.LeadTimeDays, &item.Active, &item.Version, &item.EffectiveFrom, &effectiveTo,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if effectiveTo.Valid {
			end := effectiveTo.Time
			item.EffectiveTo = &end
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSupplierContract(ctx context.Context, contract domain.SupplierContract) (domain.SupplierContract, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplierContract{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var supplierID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM suppliers WHERE id=$1 AND status='ACTIVE' FOR UPDATE`, contract.SupplierID).Scan(&supplierID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SupplierContract{}, domain.ErrInvalidState
		}
		return domain.SupplierContract{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(revision),0)+1 FROM supplier_contracts WHERE supplier_id=$1 AND contract_number=$2`, contract.SupplierID, contract.ContractNumber).Scan(&contract.Revision); err != nil {
		return domain.SupplierContract{}, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO supplier_contracts(id,supplier_id,contract_number,revision,scope,payment_terms,document_file_id,effective_from,effective_to,status,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING created_at,updated_at`, contract.ID, contract.SupplierID, contract.ContractNumber, contract.Revision, contract.Scope, contract.PaymentTerms, contract.DocumentFileID, contract.EffectiveFrom, contract.EffectiveTo, contract.Status, contract.CreatedBy).Scan(&contract.CreatedAt, &contract.UpdatedAt)
	if err != nil {
		return domain.SupplierContract{}, mapError(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO supplier_contract_status_history(id,supplier_contract_id,to_status,action,actor_id) VALUES($1,$2,$3,'CREATED',$4)`, uuid.New(), contract.ID, contract.Status, contract.CreatedBy); err != nil {
		return domain.SupplierContract{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupplierContract{}, err
	}
	return contract, nil
}

func (r *Repository) GetSupplierContract(ctx context.Context, id uuid.UUID) (domain.SupplierContract, error) {
	return scanSupplierContract(r.pool.QueryRow(ctx, supplierContractSelect+` WHERE id=$1`, id))
}

func (r *Repository) ListSupplierContracts(ctx context.Context, supplierID uuid.UUID, limit, offset int) ([]domain.SupplierContract, error) {
	var filter any
	if supplierID != uuid.Nil {
		filter = supplierID
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM supplier_contracts WHERE ($1::uuid IS NULL OR supplier_id=$1) ORDER BY supplier_id,contract_number,revision DESC LIMIT $2 OFFSET $3`, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]domain.SupplierContract, 0, len(ids))
	for _, id := range ids {
		item, err := r.GetSupplierContract(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) SetSupplierContractStatus(ctx context.Context, id uuid.UUID, status domain.SupplierContractStatus, actor string) (domain.SupplierContract, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplierContract{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanSupplierContract(tx.QueryRow(ctx, supplierContractSelect+` WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return domain.SupplierContract{}, err
	}
	if status == domain.SupplierContractActive {
		var supplier uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM suppliers WHERE id=$1 AND status='ACTIVE' FOR UPDATE`, current.SupplierID).Scan(&supplier); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.SupplierContract{}, domain.ErrInvalidState
			}
			return domain.SupplierContract{}, err
		}
		if current.Status != domain.SupplierContractDraft || current.EffectiveTo != nil && !time.Now().UTC().Before(*current.EffectiveTo) {
			return domain.SupplierContract{}, domain.ErrInvalidState
		}
		var overlap bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_contracts WHERE supplier_id=$1 AND lower(scope)=lower($2) AND status='ACTIVE' AND id<>$3 AND effective_from<COALESCE($5::timestamptz,'infinity') AND COALESCE(effective_to,'infinity')>$4::timestamptz)`, current.SupplierID, current.Scope, current.ID, current.EffectiveFrom, current.EffectiveTo).Scan(&overlap); err != nil {
			return domain.SupplierContract{}, err
		}
		if overlap {
			return domain.SupplierContract{}, domain.ErrConflict
		}
	} else if status == domain.SupplierContractTerminated {
		if current.Status != domain.SupplierContractActive {
			return domain.SupplierContract{}, domain.ErrInvalidState
		}
	} else {
		return domain.SupplierContract{}, domain.ErrInvalidArgument
	}
	previous := current.Status
	current.Status = status
	if status == domain.SupplierContractExpired {
		return domain.SupplierContract{}, domain.ErrInvalidArgument
	}
	if err := tx.QueryRow(ctx, `UPDATE supplier_contracts SET status=$2,updated_at=now() WHERE id=$1 RETURNING updated_at`, id, status).Scan(&current.UpdatedAt); err != nil {
		return domain.SupplierContract{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO supplier_contract_status_history(id,supplier_contract_id,from_status,to_status,action,actor_id) VALUES($1,$2,$3,$4,$5,$6)`, uuid.New(), id, previous, status, string(status), actor); err != nil {
		return domain.SupplierContract{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupplierContract{}, err
	}
	return current, nil
}

func (r *Repository) CreatePurchaseOrder(ctx context.Context, order domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const query = `INSERT INTO purchase_orders
		(id, po_number, supplier_id, supplier_name, status, currency, total_amount_minor, notes,
		expected_delivery_date, created_by, version, revision)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, query, order.ID, order.Number, order.SupplierID, order.SupplierName, order.Status,
		order.Currency, order.TotalAmountMinor, order.Notes, order.ExpectedDeliveryDate, order.CreatedBy, order.Version, order.Revision).
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
			&line.Quantity, &line.ReceivedQuantity, &line.AcceptedQuantity, &line.Unit, &line.UnitPriceMinor, &line.LineAmountMinor); err != nil {
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
		sent_by=$17, sent_at=$18, supplier_response_note=$19, supplier_responded_by=$20, supplier_responded_at=$21,
		cancelled_by=$22, cancelled_at=$23, cancellation_reason=$24, closed_by=$25, closed_at=$26, close_reason=$27, revision=$28,
		updated_at=now(), version=version+1
		WHERE id=$1 AND version=$2
		RETURNING created_at, updated_at, version`
	err = tx.QueryRow(ctx, query, order.ID, expectedVersion, order.SupplierID, order.SupplierName, order.Status,
		order.Currency, order.TotalAmountMinor, order.Notes, order.ExpectedDeliveryDate,
		emptyToNil(order.SubmittedBy), order.SubmittedAt, emptyToNil(order.ApprovedBy), order.ApprovedAt,
		emptyToNil(order.RejectedBy), order.RejectedAt, order.RejectionReason,
		emptyToNil(order.SentBy), order.SentAt, order.SupplierResponseNote, emptyToNil(order.SupplierRespondedBy), order.SupplierRespondedAt,
		emptyToNil(order.CancelledBy), order.CancelledAt, order.CancellationReason, emptyToNil(order.ClosedBy), order.ClosedAt, order.CloseReason, order.Revision).
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

func (r *Repository) RevisePurchaseOrder(ctx context.Context, order domain.PurchaseOrder, expectedVersion int64, actor, reason string) (domain.PurchaseOrder, error) {
	previous, err := r.GetPurchaseOrder(ctx, order.ID)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if previous.Version != expectedVersion {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	encoded, err := json.Marshal(previous)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion int64
	var currentStatus domain.PurchaseOrderStatus
	if err := tx.QueryRow(ctx, `SELECT version,status FROM purchase_orders WHERE id=$1 FOR UPDATE`, order.ID).Scan(&currentVersion, &currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PurchaseOrder{}, domain.ErrNotFound
		}
		return domain.PurchaseOrder{}, err
	}
	if currentVersion != expectedVersion || currentStatus != domain.PurchaseOrderChangeRequested {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	var hasReceipt bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM goods_receipts WHERE purchase_order_id=$1)`, order.ID).Scan(&hasReceipt); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if hasReceipt {
		return domain.PurchaseOrder{}, domain.ErrInvalidState
	}
	for _, line := range previous.Lines {
		if line.ReceivedQuantity != "0" && line.ReceivedQuantity != "0.000000" {
			q, ok := new(big.Rat).SetString(line.ReceivedQuantity)
			if ok && q.Sign() > 0 {
				return domain.PurchaseOrder{}, domain.ErrInvalidState
			}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO purchase_order_revision_snapshots(purchase_order_id,revision,snapshot,changed_by,change_reason) VALUES($1,$2,$3,$4,$5)`, order.ID, previous.Revision, encoded, actor, reason); err != nil {
		return domain.PurchaseOrder{}, mapError(err)
	}
	const query = `UPDATE purchase_orders SET supplier_id=$3,supplier_name=$4,status=$5,currency=$6,total_amount_minor=$7,notes=$8,expected_delivery_date=$9,created_by=$10,
		submitted_by=NULL,submitted_at=NULL,approved_by=NULL,approved_at=NULL,rejected_by=NULL,rejected_at=NULL,rejection_reason='',sent_by=NULL,sent_at=NULL,
		supplier_response_note='',supplier_responded_by=NULL,supplier_responded_at=NULL,cancelled_by=NULL,cancelled_at=NULL,cancellation_reason='',closed_by=NULL,closed_at=NULL,close_reason='',revision=$11,
		updated_at=now(),version=version+1 WHERE id=$1 AND version=$2 RETURNING created_at,updated_at,version`
	err = tx.QueryRow(ctx, query, order.ID, expectedVersion, order.SupplierID, order.SupplierName, order.Status, order.Currency, order.TotalAmountMinor, order.Notes, order.ExpectedDeliveryDate, order.CreatedBy, order.Revision).Scan(&order.CreatedAt, &order.UpdatedAt, &order.Version)
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
	if err := insertPurchaseOrderHistory(ctx, tx, order.ID, string(currentStatus), string(order.Status), "REVISION_CREATED", actor, reason); err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return order, nil
}

func (r *Repository) ListPurchaseOrderRevisions(ctx context.Context, id uuid.UUID, limit, offset int) ([]domain.PurchaseOrderRevision, error) {
	rows, err := r.pool.Query(ctx, `SELECT revision,snapshot,changed_by,change_reason,created_at FROM purchase_order_revision_snapshots WHERE purchase_order_id=$1 ORDER BY revision DESC LIMIT $2 OFFSET $3`, id, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.PurchaseOrderRevision, 0)
	for rows.Next() {
		var item domain.PurchaseOrderRevision
		var snapshot []byte
		if err := rows.Scan(&item.Revision, &snapshot, &item.ChangedBy, &item.ChangeReason, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &item.Snapshot); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSupplyPlan(ctx context.Context, plan domain.SupplyPlan) (domain.SupplyPlan, error) {
	snapshot, err := json.Marshal(plan.Snapshot)
	if err != nil {
		return domain.SupplyPlan{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplyPlan{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `INSERT INTO supply_plans (id,plan_number,status,current_revision,created_by,version)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at`, plan.ID, plan.Number, plan.Status, plan.Revision, plan.CreatedBy, plan.Version).
		Scan(&plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return domain.SupplyPlan{}, mapError(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO supply_plan_revisions (supply_plan_id,revision,snapshot,created_by) VALUES ($1,$2,$3,$4)`, plan.ID, plan.Revision, snapshot, plan.CreatedBy); err != nil {
		return domain.SupplyPlan{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupplyPlan{}, err
	}
	return plan, nil
}

func (r *Repository) GetSupplyPlan(ctx context.Context, id uuid.UUID) (domain.SupplyPlan, error) {
	return scanSupplyPlan(r.pool.QueryRow(ctx, supplyPlanSelect+` WHERE plan.id=$1`, id))
}

func (r *Repository) ListSupplyPlans(ctx context.Context, limit, offset int) ([]domain.SupplyPlan, error) {
	rows, err := r.pool.Query(ctx, supplyPlanSelect+` ORDER BY plan.created_at DESC,plan.id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.SupplyPlan, 0)
	for rows.Next() {
		item, scanErr := scanSupplyPlan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const supplyPlanSelect = `SELECT plan.id,plan.plan_number,plan.current_revision,plan.status,revision.snapshot,plan.created_by,plan.created_at,plan.updated_at,plan.version
	FROM supply_plans AS plan JOIN supply_plan_revisions AS revision ON revision.supply_plan_id=plan.id AND revision.revision=plan.current_revision`

func scanSupplyPlan(row rowScanner) (domain.SupplyPlan, error) {
	var plan domain.SupplyPlan
	var status string
	var snapshot []byte
	err := row.Scan(&plan.ID, &plan.Number, &plan.Revision, &status, &snapshot, &plan.CreatedBy, &plan.CreatedAt, &plan.UpdatedAt, &plan.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SupplyPlan{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SupplyPlan{}, err
	}
	plan.Status = domain.SupplyPlanStatus(status)
	if err := json.Unmarshal(snapshot, &plan.Snapshot); err != nil {
		return domain.SupplyPlan{}, err
	}
	return plan, nil
}

func (r *Repository) CreateGoodsReceipt(ctx context.Context, receipt domain.GoodsReceipt) (domain.GoodsReceipt, error) {
	if receipt.IdempotencyKey != "" {
		var existingID uuid.UUID
		var existingHash string
		err := r.pool.QueryRow(ctx, `SELECT id,idempotency_request_hash FROM goods_receipts WHERE received_by=$1 AND idempotency_key=$2`, receipt.ReceivedBy, receipt.IdempotencyKey).Scan(&existingID, &existingHash)
		if err == nil {
			if existingHash != receipt.RequestHash {
				return domain.GoodsReceipt{}, domain.ErrConflict
			}
			return r.GetGoodsReceipt(ctx, existingID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.GoodsReceipt{}, err
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status domain.PurchaseOrderStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1 FOR UPDATE`, receipt.PurchaseOrderID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GoodsReceipt{}, domain.ErrNotFound
		}
		return domain.GoodsReceipt{}, err
	}
	if status != domain.PurchaseOrderSent && status != domain.PurchaseOrderAcknowledged && status != domain.PurchaseOrderPartiallyReceived {
		return domain.GoodsReceipt{}, domain.ErrInvalidState
	}
	for _, line := range receipt.Lines {
		var lineID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM purchase_order_lines WHERE id=$1 AND purchase_order_id=$2 FOR UPDATE`, line.PurchaseOrderLineID, receipt.PurchaseOrderID).Scan(&lineID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.GoodsReceipt{}, domain.ErrNotFound
			}
			return domain.GoodsReceipt{}, err
		}
	}
	encoded, err := json.Marshal(receipt.Lines)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	variancesEncoded, err := json.Marshal(receipt.Variances)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	insertErr := tx.QueryRow(ctx, `INSERT INTO goods_receipts (id, receipt_number, purchase_order_id, purchase_order_number, supplier_id, supplier_name, warehouse_code, status, lines, variances, received_by, notes, version, idempotency_key, idempotency_request_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT (received_by,idempotency_key) DO NOTHING RETURNING created_at,updated_at`, receipt.ID, receipt.Number, receipt.PurchaseOrderID, receipt.PurchaseOrderNumber,
		receipt.SupplierID, receipt.SupplierName, receipt.WarehouseCode, receipt.Status, encoded, variancesEncoded, receipt.ReceivedBy, receipt.Notes, receipt.Version, emptyToNil(receipt.IdempotencyKey), emptyToNil(receipt.RequestHash)).Scan(&receipt.CreatedAt, &receipt.UpdatedAt)
	if errors.Is(insertErr, pgx.ErrNoRows) && receipt.IdempotencyKey != "" {
		_ = tx.Rollback(ctx)
		var existingID uuid.UUID
		var existingHash string
		if err := r.pool.QueryRow(ctx, `SELECT id,idempotency_request_hash FROM goods_receipts WHERE received_by=$1 AND idempotency_key=$2`, receipt.ReceivedBy, receipt.IdempotencyKey).Scan(&existingID, &existingHash); err != nil {
			return domain.GoodsReceipt{}, err
		}
		if existingHash != receipt.RequestHash {
			return domain.GoodsReceipt{}, domain.ErrConflict
		}
		return r.GetGoodsReceipt(ctx, existingID)
	}
	if insertErr != nil {
		return domain.GoodsReceipt{}, mapError(insertErr)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO goods_receipt_status_history (id,goods_receipt_id,to_status,action,actor_id) VALUES ($1,$2,$3,'CREATED',$4)`, uuid.New(), receipt.ID, receipt.Status, receipt.ReceivedBy); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return receipt, nil
}

func (r *Repository) GetGoodsReceipt(ctx context.Context, id uuid.UUID) (domain.GoodsReceipt, error) {
	var receipt domain.GoodsReceipt
	var status string
	var lines, variances []byte
	var inspectedBy pgtype.Text
	var inspectedAt pgtype.Timestamptz
	err := r.pool.QueryRow(ctx, `SELECT id,receipt_number,purchase_order_id,purchase_order_number,supplier_id,supplier_name,warehouse_code,status,lines,variances,received_by,inspected_by,inspected_at,notes,created_at,updated_at,version FROM goods_receipts WHERE id=$1`, id).
		Scan(&receipt.ID, &receipt.Number, &receipt.PurchaseOrderID, &receipt.PurchaseOrderNumber, &receipt.SupplierID, &receipt.SupplierName, &receipt.WarehouseCode, &status, &lines, &variances, &receipt.ReceivedBy, &inspectedBy, &inspectedAt, &receipt.Notes, &receipt.CreatedAt, &receipt.UpdatedAt, &receipt.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	receipt.Status = domain.GoodsReceiptStatus(status)
	receipt.InspectedBy = inspectedBy.String
	receipt.InspectedAt = timestampPointer(inspectedAt)
	if err := json.Unmarshal(lines, &receipt.Lines); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := json.Unmarshal(variances, &receipt.Variances); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return receipt, nil
}

func (r *Repository) GetGoodsReceiptByIdempotency(ctx context.Context, actor, key string) (domain.GoodsReceipt, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM goods_receipts WHERE received_by=$1 AND idempotency_key=$2`, actor, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	receipt, err := r.GetGoodsReceipt(ctx, id)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := r.pool.QueryRow(ctx, `SELECT idempotency_request_hash FROM goods_receipts WHERE id=$1`, id).Scan(&receipt.RequestHash); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return receipt, nil
}

func (r *Repository) HasUnresolvedGoodsReceiptVariances(ctx context.Context, purchaseOrderID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM goods_receipts AS receipt
		CROSS JOIN LATERAL jsonb_array_elements(receipt.variances) AS variance(item)
		WHERE receipt.purchase_order_id=$1 AND variance.item->>'status'='OPEN'
	)`, purchaseOrderID).Scan(&exists)
	return exists, err
}

func (r *Repository) ResolveGoodsReceiptVariance(ctx context.Context, receiptID, varianceID uuid.UUID, decision, actor, note string) (domain.GoodsReceipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var receipt domain.GoodsReceipt
	var status string
	var lines, variances []byte
	if err := tx.QueryRow(ctx, `SELECT id,purchase_order_id,status,lines,variances,version FROM goods_receipts WHERE id=$1 FOR UPDATE`, receiptID).
		Scan(&receipt.ID, &receipt.PurchaseOrderID, &status, &lines, &variances, &receipt.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GoodsReceipt{}, domain.ErrNotFound
		}
		return domain.GoodsReceipt{}, err
	}
	receipt.Status = domain.GoodsReceiptStatus(status)
	if err := json.Unmarshal(lines, &receipt.Lines); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := json.Unmarshal(variances, &receipt.Variances); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := receipt.ResolveVariance(varianceID, decision, actor, note); err != nil {
		return domain.GoodsReceipt{}, err
	}
	encoded, err := json.Marshal(receipt.Variances)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := tx.QueryRow(ctx, `UPDATE goods_receipts SET variances=$2,updated_at=now(),version=version+1 WHERE id=$1 RETURNING updated_at,version`, receiptID, encoded).Scan(&receipt.UpdatedAt, &receipt.Version); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO goods_receipt_status_history (id,goods_receipt_id,from_status,to_status,action,actor_id) VALUES ($1,$2,$3,$3,'VARIANCE_RESOLVED',$4)`, uuid.New(), receiptID, receipt.Status, actor); err != nil {
		return domain.GoodsReceipt{}, err
	}
	var hasOpen bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM goods_receipts AS existing_receipt
		CROSS JOIN LATERAL jsonb_array_elements(existing_receipt.variances) AS variance(item)
		WHERE existing_receipt.purchase_order_id=$1 AND variance.item->>'status'='OPEN'
	)`, receipt.PurchaseOrderID).Scan(&hasOpen); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if !hasOpen {
		var nextPOStatus domain.PurchaseOrderStatus
		if err := tx.QueryRow(ctx, `SELECT CASE WHEN bool_and(accepted_quantity>=quantity) THEN 'RECEIVED' ELSE 'PARTIALLY_RECEIVED' END FROM purchase_order_lines WHERE purchase_order_id=$1`, receipt.PurchaseOrderID).Scan(&nextPOStatus); err != nil {
			return domain.GoodsReceipt{}, err
		}
		var currentPOStatus domain.PurchaseOrderStatus
		if err := tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1 FOR UPDATE`, receipt.PurchaseOrderID).Scan(&currentPOStatus); err != nil {
			return domain.GoodsReceipt{}, err
		}
		if currentPOStatus != nextPOStatus {
			if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status=$2,version=version+1,updated_at=now() WHERE id=$1`, receipt.PurchaseOrderID, nextPOStatus); err != nil {
				return domain.GoodsReceipt{}, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO purchase_order_status_history (id,purchase_order_id,from_status,to_status,action,actor_id,reason) VALUES ($1,$2,$3,$4,'VARIANCE_RESOLVED',$5,$6)`, uuid.New(), receipt.PurchaseOrderID, currentPOStatus, nextPOStatus, actor, note); err != nil {
				return domain.GoodsReceipt{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return r.GetGoodsReceipt(ctx, receiptID)
}

func (r *Repository) InspectGoodsReceipt(ctx context.Context, receipt domain.GoodsReceipt, expectedVersion int64, actor string) (domain.GoodsReceipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion int64
	var oldStatus domain.GoodsReceiptStatus
	if err := tx.QueryRow(ctx, `SELECT version,status FROM goods_receipts WHERE id=$1 FOR UPDATE`, receipt.ID).Scan(&currentVersion, &oldStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GoodsReceipt{}, domain.ErrNotFound
		}
		return domain.GoodsReceipt{}, err
	}
	if currentVersion != expectedVersion || oldStatus != domain.GoodsReceiptDraft {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	var poStatus domain.PurchaseOrderStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1 FOR UPDATE`, receipt.PurchaseOrderID).Scan(&poStatus); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if poStatus != domain.PurchaseOrderSent && poStatus != domain.PurchaseOrderAcknowledged && poStatus != domain.PurchaseOrderPartiallyReceived {
		return domain.GoodsReceipt{}, domain.ErrInvalidState
	}
	for _, line := range receipt.Lines {
		var orderedText, acceptedText string
		if err := tx.QueryRow(ctx, `SELECT quantity::text,accepted_quantity::text FROM purchase_order_lines WHERE id=$1 AND purchase_order_id=$2 FOR UPDATE`, line.PurchaseOrderLineID, receipt.PurchaseOrderID).Scan(&orderedText, &acceptedText); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.GoodsReceipt{}, domain.ErrNotFound
			}
			return domain.GoodsReceipt{}, err
		}
		ordered, orderedOK := new(big.Rat).SetString(orderedText)
		accepted, acceptedOK := new(big.Rat).SetString(acceptedText)
		received, receivedOK := new(big.Rat).SetString(line.QuantityReceived)
		if !orderedOK || !acceptedOK || !receivedOK {
			return domain.GoodsReceipt{}, domain.ErrInvalidArgument
		}
		openQuantity := new(big.Rat).Sub(ordered, accepted)
		filtered := receipt.Variances[:0]
		for _, variance := range receipt.Variances {
			if variance.GoodsReceiptLineID != line.ID || variance.Type != domain.GoodsReceiptVarianceOverReceived {
				filtered = append(filtered, variance)
			}
		}
		receipt.Variances = filtered
		if over := new(big.Rat).Sub(new(big.Rat).Set(received), openQuantity); over.Sign() > 0 {
			if strings.TrimSpace(line.InspectionNotes) == "" {
				return domain.GoodsReceipt{}, domain.ErrInvalidArgument
			}
			receipt.Variances = append(receipt.Variances, domain.GoodsReceiptVariance{ID: uuid.New(), GoodsReceiptLineID: line.ID,
				Type: domain.GoodsReceiptVarianceOverReceived, ExpectedQuantity: openQuantity.FloatString(6), ActualQuantity: received.FloatString(6),
				DifferenceQuantity: over.FloatString(6), Reason: line.InspectionNotes, Status: "OPEN"})
		}
		tag, err := tx.Exec(ctx, `UPDATE purchase_order_lines SET received_quantity=received_quantity+$1::numeric, accepted_quantity=accepted_quantity+$2::numeric WHERE id=$3 AND purchase_order_id=$4 AND accepted_quantity+$2::numeric<=quantity`, line.QuantityReceived, line.QuantityAccepted, line.PurchaseOrderLineID, receipt.PurchaseOrderID)
		if err != nil {
			return domain.GoodsReceipt{}, err
		}
		if tag.RowsAffected() != 1 {
			return domain.GoodsReceipt{}, domain.ErrConflict
		}
	}
	var nextPOStatus domain.PurchaseOrderStatus
	if err := tx.QueryRow(ctx, `SELECT CASE WHEN bool_and(accepted_quantity>=quantity) THEN 'RECEIVED' ELSE 'PARTIALLY_RECEIVED' END FROM purchase_order_lines WHERE purchase_order_id=$1`, receipt.PurchaseOrderID).Scan(&nextPOStatus); err != nil {
		return domain.GoodsReceipt{}, err
	}
	var hasEarlierOpenVariance bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM goods_receipts AS existing_receipt
		CROSS JOIN LATERAL jsonb_array_elements(existing_receipt.variances) AS variance(item)
		WHERE existing_receipt.purchase_order_id=$1 AND variance.item->>'status'='OPEN'
	)`, receipt.PurchaseOrderID).Scan(&hasEarlierOpenVariance); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if len(receipt.Variances) > 0 || hasEarlierOpenVariance {
		nextPOStatus = domain.PurchaseOrderPartiallyReceived
	}
	if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status=$2,version=version+1,updated_at=now() WHERE id=$1`, receipt.PurchaseOrderID, nextPOStatus); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO purchase_order_status_history (id,purchase_order_id,from_status,to_status,action,actor_id) VALUES ($1,$2,$3,$4,'GOODS_RECEIPT_INSPECTED',$5)`, uuid.New(), receipt.PurchaseOrderID, poStatus, nextPOStatus, actor); err != nil {
		return domain.GoodsReceipt{}, err
	}
	encoded, err := json.Marshal(receipt.Lines)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	variancesEncoded, err := json.Marshal(receipt.Variances)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	err = tx.QueryRow(ctx, `UPDATE goods_receipts SET status=$3,lines=$4,variances=$5,inspected_by=$6,inspected_at=$7,updated_at=now(),version=version+1 WHERE id=$1 AND version=$2 RETURNING updated_at,version`, receipt.ID, expectedVersion, receipt.Status, encoded, variancesEncoded, receipt.InspectedBy, receipt.InspectedAt).Scan(&receipt.UpdatedAt, &receipt.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO goods_receipt_status_history (id,goods_receipt_id,from_status,to_status,action,actor_id) VALUES ($1,$2,$3,$4,'INSPECTED',$5)`, uuid.New(), receipt.ID, oldStatus, receipt.Status, actor); err != nil {
		return domain.GoodsReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return receipt, nil
}

func insertPurchaseOrderLines(ctx context.Context, tx pgx.Tx, order domain.PurchaseOrder) error {
	const query = `INSERT INTO purchase_order_lines
		(id, purchase_order_id, line_number, material_id, material_code, material_name, quantity, received_quantity, accepted_quantity, unit, unit_price_minor, line_amount_minor)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	for _, line := range order.Lines {
		receivedQuantity := line.ReceivedQuantity
		if receivedQuantity == "" {
			receivedQuantity = "0"
		}
		acceptedQuantity := line.AcceptedQuantity
		if acceptedQuantity == "" {
			acceptedQuantity = "0"
		}
		if _, err := tx.Exec(ctx, query, line.ID, order.ID, line.LineNumber, line.MaterialID, line.MaterialCode,
			line.MaterialName, line.Quantity, receivedQuantity, acceptedQuantity, line.Unit, line.UnitPriceMinor, line.LineAmountMinor); err != nil {
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
	revision, total_amount_minor, notes, expected_delivery_date, created_by, submitted_by, submitted_at,
	approved_by, approved_at, rejected_by, rejected_at, rejection_reason, sent_by, sent_at,
	supplier_response_note,supplier_responded_by,supplier_responded_at,cancelled_by,cancelled_at,cancellation_reason,
	closed_by,closed_at,close_reason,created_at,updated_at,version FROM purchase_orders`

const purchaseOrderLineSelect = `SELECT id, line_number, material_id, material_code, material_name,
	quantity::text, received_quantity::text, accepted_quantity::text, unit, unit_price_minor, line_amount_minor FROM purchase_order_lines`

func scanPurchaseOrder(row rowScanner) (domain.PurchaseOrder, error) {
	var order domain.PurchaseOrder
	var expected pgtype.Date
	var submitted, approved, rejected, sent, supplierResponded, cancelled, closed pgtype.Timestamptz
	var submittedBy, approvedBy, rejectedBy, rejectionReason, sentBy, supplierRespondedBy, cancellationReason, cancelledBy, closedBy, closeReason pgtype.Text
	err := row.Scan(&order.ID, &order.Number, &order.SupplierID, &order.SupplierName, &order.Status, &order.Currency,
		&order.Revision, &order.TotalAmountMinor, &order.Notes, &expected, &order.CreatedBy, &submittedBy, &submitted,
		&approvedBy, &approved, &rejectedBy, &rejected, &rejectionReason, &sentBy, &sent,
		&order.SupplierResponseNote, &supplierRespondedBy, &supplierResponded, &cancelledBy, &cancelled, &cancellationReason,
		&closedBy, &closed, &closeReason, &order.CreatedAt, &order.UpdatedAt, &order.Version)
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
	order.SupplierRespondedBy, order.CancellationReason = supplierRespondedBy.String, cancellationReason.String
	order.ClosedBy, order.CloseReason = closedBy.String, closeReason.String
	order.SubmittedAt = timestampPointer(submitted)
	order.ApprovedAt = timestampPointer(approved)
	order.RejectedAt = timestampPointer(rejected)
	order.SentAt = timestampPointer(sent)
	order.SupplierRespondedAt = timestampPointer(supplierResponded)
	order.CancelledAt = timestampPointer(cancelled)
	order.ClosedAt = timestampPointer(closed)
	return order, nil
}

func timestampPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

const supplierSelect = `SELECT id, code, name, supplier_type, contact_name, phone, email, address, legal_identifier, status, created_at, updated_at FROM suppliers`
const materialSelect = `SELECT id, code, name, base_unit, description, status, preferred_supplier_id, created_at, updated_at FROM raw_materials`
const supplierContractSelect = `SELECT id,supplier_id,contract_number,revision,scope,payment_terms,document_file_id,effective_from,effective_to,status,created_by,created_at,updated_at FROM supplier_contracts`

type rowScanner interface{ Scan(...any) error }

func scanSupplier(row rowScanner) (domain.Supplier, error) {
	var item domain.Supplier
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.SupplierType, &item.ContactName, &item.Phone,
		&item.Email, &item.Address, &item.LegalIdentifier, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Supplier{}, domain.ErrNotFound
	}
	return item, err
}

func scanSupplierRow(row rowScanner) (domain.Supplier, error) { return scanSupplier(row) }

func scanMaterial(row rowScanner) (domain.RawMaterial, error) {
	var item domain.RawMaterial
	var preferred pgtype.UUID
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.BaseUnit, &item.Description, &item.Status, &preferred, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	if preferred.Valid {
		id, parseErr := uuid.FromBytes(preferred.Bytes[:])
		if parseErr != nil {
			return domain.RawMaterial{}, parseErr
		}
		item.PreferredSupplierID = &id
	}
	return item, err
}

func scanMaterialRow(row rowScanner) (domain.RawMaterial, error) { return scanMaterial(row) }

func scanSupplierContract(row rowScanner) (domain.SupplierContract, error) {
	var item domain.SupplierContract
	var effectiveTo pgtype.Timestamptz
	err := row.Scan(&item.ID, &item.SupplierID, &item.ContractNumber, &item.Revision, &item.Scope, &item.PaymentTerms, &item.DocumentFileID, &item.EffectiveFrom, &effectiveTo, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SupplierContract{}, domain.ErrNotFound
	}
	if effectiveTo.Valid {
		end := effectiveTo.Time
		item.EffectiveTo = &end
	}
	return item, err
}

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
