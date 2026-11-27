package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/omamx/procurement-service/internal/domain"
)

// Low-volume master commands serialize before row locks. Price-version creates
// and legacy multi-row status operations use the same ordering.
func lockMasterData(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('procurement-master-data-write',0))`)
	return err
}
func getMasterMutationResult(ctx context.Context, q rowQuerier, actor, key string) (domain.MasterDataMutationResult, error) {
	var result domain.MasterDataMutationResult
	err := q.QueryRow(ctx, `SELECT request_hash,result_snapshot FROM master_data_mutation_results WHERE actor_id=$1 AND idempotency_key=$2`, actor, key).Scan(&result.RequestHash, &result.Snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, domain.ErrNotFound
	}
	return result, err
}
func (r *Repository) GetMasterDataMutationResult(ctx context.Context, actor, key string) (domain.MasterDataMutationResult, error) {
	return getMasterMutationResult(ctx, r.pool, actor, key)
}

const masterOfferSelect = `SELECT id,supplier_id,material_id,supplier_sku,unit,unit_price_minor,currency,minimum_quantity::text,lead_time_days,active,version,effective_from,effective_to,created_at,updated_at FROM supplier_material_offers`

func scanMasterOffer(row rowScanner) (domain.SupplierMaterialOffer, error) {
	var v domain.SupplierMaterialOffer
	var end pgtype.Timestamptz
	err := row.Scan(&v.ID, &v.SupplierID, &v.MaterialID, &v.SupplierSKU, &v.Unit, &v.UnitPriceMinor, &v.Currency, &v.MinimumQty, &v.LeadTimeDays, &v.Active, &v.Version, &v.EffectiveFrom, &end, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.ErrNotFound
	}
	if err != nil {
		return v, mapError(err)
	}
	if end.Valid {
		v.EffectiveTo = &end.Time
	}
	return v, nil
}

func (r *Repository) MutateMasterData(ctx context.Context, cmd domain.MasterDataMutation) (domain.MasterDataMutationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.MasterDataMutationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockMasterData(ctx, tx); err != nil {
		return domain.MasterDataMutationResult{}, err
	}
	prior, err := getMasterMutationResult(ctx, tx, cmd.Actor, cmd.Key)
	if err == nil {
		if prior.RequestHash != cmd.RequestHash {
			return prior, domain.ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MasterDataMutationResult{}, err
	}
	resource, err := mutateMasterResource(ctx, tx, cmd)
	if err != nil {
		return domain.MasterDataMutationResult{}, mapError(err)
	}
	snapshot, err := json.Marshal(resource)
	if err != nil {
		return domain.MasterDataMutationResult{}, err
	}
	var parent any
	if cmd.Action == "OFFER_STATUS" {
		parent = cmd.SupplierID
	}
	if _, err = tx.Exec(ctx, `INSERT INTO master_data_mutation_results(actor_id,idempotency_key,action,resource_id,parent_supplier_id,expected_updated_at,request_hash,result_snapshot) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, cmd.Actor, cmd.Key, cmd.Action, cmd.ResourceID, parent, cmd.ExpectedUpdatedAt, cmd.RequestHash, snapshot); err != nil {
		return domain.MasterDataMutationResult{}, mapError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.MasterDataMutationResult{}, err
	}
	return domain.MasterDataMutationResult{RequestHash: cmd.RequestHash, Snapshot: snapshot}, nil
}

func mutateMasterResource(ctx context.Context, tx pgx.Tx, cmd domain.MasterDataMutation) (any, error) {
	switch cmd.Action {
	case "EDIT_SUPPLIER", "SUPPLIER_STATUS":
		current, err := scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE id=$1 FOR UPDATE`, cmd.ResourceID))
		if err != nil {
			return nil, err
		}
		if !current.UpdatedAt.Equal(cmd.ExpectedUpdatedAt) {
			return nil, domain.ErrConflict
		}
		if cmd.Action == "EDIT_SUPPLIER" {
			v := cmd.Supplier
			if v.Code != current.Code {
				return nil, domain.ErrConflict
			}
			_, err = tx.Exec(ctx, `UPDATE suppliers SET name=$2,supplier_type=$3,contact_name=$4,phone=$5,email=$6,address=$7,legal_identifier=$8 WHERE id=$1`, cmd.ResourceID, v.Name, v.SupplierType, v.ContactName, v.Phone, v.Email, v.Address, v.LegalIdentifier)
		} else {
			_, err = tx.Exec(ctx, `UPDATE suppliers SET status=$2 WHERE id=$1`, cmd.ResourceID, cmd.Status)
			if err == nil && cmd.Status == "SUSPENDED" {
				_, err = tx.Exec(ctx, `UPDATE raw_materials SET preferred_supplier_id=NULL WHERE preferred_supplier_id=$1`, cmd.ResourceID)
			}
		}
		if err != nil {
			return nil, err
		}
		return scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE id=$1`, cmd.ResourceID))
	case "EDIT_MATERIAL", "MATERIAL_STATUS", "PREFERRED_SUPPLIER":
		if cmd.Action == "PREFERRED_SUPPLIER" && cmd.PreferredSupplierID != uuid.Nil {
			supplier, err := scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE id=$1 FOR UPDATE`, cmd.PreferredSupplierID))
			if err != nil {
				return nil, err
			}
			if supplier.Status != domain.SupplierActive {
				return nil, domain.ErrInvalidState
			}
		}
		current, err := scanMaterial(tx.QueryRow(ctx, materialSelect+` WHERE id=$1 FOR UPDATE`, cmd.ResourceID))
		if err != nil {
			return nil, err
		}
		if !current.UpdatedAt.Equal(cmd.ExpectedUpdatedAt) {
			return nil, domain.ErrConflict
		}
		switch cmd.Action {
		case "EDIT_MATERIAL":
			v := cmd.Material
			if v.Code != current.Code || v.BaseUnit != current.BaseUnit {
				return nil, domain.ErrConflict
			}
			_, err = tx.Exec(ctx, `UPDATE raw_materials SET name=$2,description=$3 WHERE id=$1`, cmd.ResourceID, v.Name, v.Description)
		case "MATERIAL_STATUS":
			_, err = tx.Exec(ctx, `UPDATE raw_materials SET status=$2::varchar,preferred_supplier_id=CASE WHEN $2::varchar='INACTIVE' THEN NULL ELSE preferred_supplier_id END WHERE id=$1`, cmd.ResourceID, cmd.Status)
		case "PREFERRED_SUPPLIER":
			var preferred any
			if cmd.PreferredSupplierID != uuid.Nil {
				if current.Status != domain.MaterialActive {
					return nil, domain.ErrInvalidState
				}
				var eligible bool
				if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2 AND unit=$3 AND active AND effective_from<=clock_timestamp() AND (effective_to IS NULL OR effective_to>clock_timestamp()))`, cmd.PreferredSupplierID, cmd.ResourceID, current.BaseUnit).Scan(&eligible); err != nil {
					return nil, err
				}
				if !eligible {
					return nil, domain.ErrInvalidState
				}
				preferred = cmd.PreferredSupplierID
			}
			_, err = tx.Exec(ctx, `UPDATE raw_materials SET preferred_supplier_id=$2 WHERE id=$1`, cmd.ResourceID, preferred)
		}
		if err != nil {
			return nil, err
		}
		return scanMaterial(tx.QueryRow(ctx, materialSelect+` WHERE id=$1`, cmd.ResourceID))
	case "OFFER_STATUS":
		supplier, err := scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE id=$1 FOR UPDATE`, cmd.SupplierID))
		if err != nil {
			return nil, err
		}
		offer, err := scanMasterOffer(tx.QueryRow(ctx, masterOfferSelect+` WHERE id=$1 AND supplier_id=$2 FOR UPDATE`, cmd.ResourceID, cmd.SupplierID))
		if err != nil {
			return nil, err
		}
		if !offer.UpdatedAt.Equal(cmd.ExpectedUpdatedAt) {
			return nil, domain.ErrConflict
		}
		material, err := scanMaterial(tx.QueryRow(ctx, materialSelect+` WHERE id=$1 FOR UPDATE`, offer.MaterialID))
		if err != nil {
			return nil, err
		}
		if *cmd.Active {
			if supplier.Status != domain.SupplierActive || material.Status != domain.MaterialActive || offer.Unit != material.BaseUnit {
				return nil, domain.ErrInvalidState
			}
			var overlaps bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_material_offers WHERE supplier_id=$1 AND material_id=$2 AND id<>$3 AND active AND effective_from<COALESCE($5::timestamptz,'infinity') AND COALESCE(effective_to,'infinity')>$4::timestamptz)`, offer.SupplierID, offer.MaterialID, offer.ID, offer.EffectiveFrom, offer.EffectiveTo).Scan(&overlaps)
			if err != nil {
				return nil, err
			}
			if overlaps {
				return nil, domain.ErrConflict
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE supplier_material_offers SET active=$2 WHERE id=$1`, cmd.ResourceID, *cmd.Active); err != nil {
			return nil, err
		}
		if !*cmd.Active {
			_, err = tx.Exec(ctx, `UPDATE raw_materials m SET preferred_supplier_id=NULL WHERE id=$1 AND preferred_supplier_id=$2 AND NOT EXISTS(SELECT 1 FROM supplier_material_offers WHERE material_id=$1 AND supplier_id=$2 AND unit=m.base_unit AND active AND effective_from<=clock_timestamp() AND (effective_to IS NULL OR effective_to>clock_timestamp()))`, offer.MaterialID, offer.SupplierID)
			if err != nil {
				return nil, err
			}
		}
		return scanMasterOffer(tx.QueryRow(ctx, masterOfferSelect+` WHERE id=$1`, cmd.ResourceID))
	default:
		return nil, domain.ErrInvalidArgument
	}
}
