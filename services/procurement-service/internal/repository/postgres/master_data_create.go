package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/procurement-service/internal/domain"
)

func getMasterDataCreateResult(ctx context.Context, queryer rowQuerier, actor, key string) (domain.MasterDataCreateResult, error) {
	var result domain.MasterDataCreateResult
	err := queryer.QueryRow(ctx, `SELECT kind,request_hash,result_snapshot FROM master_data_create_results WHERE actor_id=$1 AND idempotency_key=$2`, actor, key).Scan(&result.Kind, &result.RequestHash, &result.Snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, domain.ErrNotFound
	}
	return result, err
}
func (r *Repository) GetMasterDataCreateResult(ctx context.Context, actor, key string) (domain.MasterDataCreateResult, error) {
	return getMasterDataCreateResult(ctx, r.pool, actor, key)
}

func (r *Repository) CreateMasterDataCommand(ctx context.Context, cmd domain.MasterDataCreateCommand) (domain.MasterDataCreateResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Lock the shared actor/key before any business row, then re-read evidence.
	// Hash collisions only serialize unrelated commands; uniqueness uses full keys.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "procurement-master-create:"+cmd.Actor+"\x1f"+cmd.Key); err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	prior, err := getMasterDataCreateResult(ctx, tx, cmd.Actor, cmd.Key)
	if err == nil {
		if prior.Kind != cmd.Kind || prior.RequestHash != cmd.RequestHash {
			return prior, domain.ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MasterDataCreateResult{}, err
	}
	var resource any
	var id uuid.UUID
	switch cmd.Kind {
	case "SUPPLIER":
		if cmd.Supplier == nil {
			return domain.MasterDataCreateResult{}, domain.ErrInvalidArgument
		}
		item, e := createSupplier(ctx, tx, *cmd.Supplier)
		err = e
		resource, id = item, item.ID
	case "MATERIAL":
		if cmd.Material == nil {
			return domain.MasterDataCreateResult{}, domain.ErrInvalidArgument
		}
		item, e := createMaterial(ctx, tx, *cmd.Material)
		err = e
		resource, id = item, item.ID
	case "OFFER":
		if cmd.Offer == nil {
			return domain.MasterDataCreateResult{}, domain.ErrInvalidArgument
		}
		item, e := createOffer(ctx, tx, *cmd.Offer)
		err = e
		resource, id = item, item.ID
	default:
		return domain.MasterDataCreateResult{}, domain.ErrInvalidArgument
	}
	if err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	snapshot, err := json.Marshal(resource)
	if err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO master_data_create_results(actor_id,idempotency_key,kind,resource_id,request_hash,result_snapshot) VALUES($1,$2,$3,$4,$5,$6)`, cmd.Actor, cmd.Key, cmd.Kind, id, cmd.RequestHash, snapshot); err != nil {
		return domain.MasterDataCreateResult{}, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.MasterDataCreateResult{}, err
	}
	return domain.MasterDataCreateResult{Kind: cmd.Kind, RequestHash: cmd.RequestHash, Snapshot: snapshot}, nil
}
