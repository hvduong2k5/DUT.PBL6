package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/omamx/procurement-service/internal/domain"
)

func getContractCommandResult(ctx context.Context, q rowQuerier, actor, key string) (domain.SupplierContractCommandResult, error) {
	var result domain.SupplierContractCommandResult
	err := q.QueryRow(ctx, `SELECT request_hash,result_snapshot FROM supplier_contract_command_results WHERE actor_id=$1 AND idempotency_key=$2`, actor, key).Scan(&result.RequestHash, &result.Snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, domain.ErrNotFound
	}
	return result, err
}
func (r *Repository) GetSupplierContractCommandResult(ctx context.Context, actor, key string) (domain.SupplierContractCommandResult, error) {
	return getContractCommandResult(ctx, r.pool, actor, key)
}
func (r *Repository) ExecuteSupplierContractCommand(ctx context.Context, c domain.SupplierContractCommand) (domain.SupplierContractCommandResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.SupplierContractCommandResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Same lock ordering as supplier status, price and other master commands.
	if err = lockMasterData(ctx, tx); err != nil {
		return domain.SupplierContractCommandResult{}, err
	}
	prior, err := getContractCommandResult(ctx, tx, c.Actor, c.Key)
	if err == nil {
		if prior.RequestHash != c.RequestHash {
			return prior, domain.ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.SupplierContractCommandResult{}, err
	}
	var contract domain.SupplierContract
	if c.Action == "CREATE" {
		if c.Contract == nil {
			return domain.SupplierContractCommandResult{}, domain.ErrInvalidArgument
		}
		contract, err = createSupplierContract(ctx, tx, *c.Contract)
	} else {
		contract, err = scanSupplierContract(tx.QueryRow(ctx, supplierContractSelect+` WHERE id=$1 FOR UPDATE`, c.ContractID))
		if err == nil {
			if !contract.UpdatedAt.Equal(c.ExpectedUpdatedAt) {
				return domain.SupplierContractCommandResult{}, domain.ErrConflict
			}
			var status domain.SupplierContractStatus
			switch c.Action {
			case "ACTIVATE":
				status = domain.SupplierContractActive
			case "TERMINATE":
				status = domain.SupplierContractTerminated
			default:
				return domain.SupplierContractCommandResult{}, domain.ErrInvalidArgument
			}
			contract, err = changeSupplierContractStatus(ctx, tx, c.ContractID, status, c.Actor, c.Reason)
		}
	}
	if err != nil {
		return domain.SupplierContractCommandResult{}, mapError(err)
	}
	snapshot, err := json.Marshal(contract)
	if err != nil {
		return domain.SupplierContractCommandResult{}, err
	}
	var expected any
	if c.Action != "CREATE" {
		expected = c.ExpectedUpdatedAt
	}
	if _, err = tx.Exec(ctx, `INSERT INTO supplier_contract_command_results(actor_id,idempotency_key,action,supplier_contract_id,expected_updated_at,reason,request_hash,result_snapshot) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, c.Actor, c.Key, c.Action, contract.ID, expected, c.Reason, c.RequestHash, snapshot); err != nil {
		return domain.SupplierContractCommandResult{}, mapError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.SupplierContractCommandResult{}, err
	}
	return domain.SupplierContractCommandResult{RequestHash: c.RequestHash, Snapshot: snapshot}, nil
}
