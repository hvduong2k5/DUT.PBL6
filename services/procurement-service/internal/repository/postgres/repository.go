package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
