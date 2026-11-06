package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrLoyaltyHoldNotFound = errors.New("loyalty redemption hold not found")
	ErrLoyaltyHoldConflict = errors.New("loyalty redemption hold conflicts with the current order state")
)

type LoyaltyHold struct {
	ID             string
	OrderID        string
	CustomerID     string
	Points         int64
	DiscountAmount int64
	Currency       string
	State          string
	Version        int64
	ExpiresAt      time.Time
	ObservedAt     time.Time
	PolicyVersion  string
	CommandResult  string
	ReasonCode     string
	Replay         bool
	Tombstone      bool
}

func (r *LoyaltyRepository) LockRedemptionPoints(ctx context.Context, orderID, customerID, key string, points int64) (LoyaltyHold, error) {
	if orderID == "" || customerID == "" || points <= 0 || !validLoyaltyIdempotencyKey(key) {
		return LoyaltyHold{}, errors.New("order, customer, positive points and valid idempotency key are required")
	}
	hash := loyaltyPayloadHash(struct {
		Action, OrderID, CustomerID string
		Points                      int64
	}{"LockRedemptionPoints", orderID, customerID, points})
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LoyaltyHold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockLoyaltyCommand(ctx, tx, key); err != nil {
		return LoyaltyHold{}, err
	}
	if hold, found, err := replayLoyaltyCommand(ctx, tx, key, "LOCK_REDEMPTION_POINTS", hash); err != nil {
		return LoyaltyHold{}, err
	} else if found {
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return LoyaltyHold{}, err
		}
		return hold, nil
	}
	if err := lockLoyaltyOrder(ctx, tx, orderID); err != nil {
		return LoyaltyHold{}, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM loyalty_redemption_holds WHERE order_id=$1)`, orderID).Scan(&exists); err != nil {
		return LoyaltyHold{}, err
	}
	if exists {
		return LoyaltyHold{}, ErrLoyaltyHoldConflict
	}
	if err := lockLoyaltyCustomer(ctx, tx, customerID); err != nil {
		return LoyaltyHold{}, err
	}
	var balance, reserved int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(points_delta),0)::bigint FROM loyalty_ledger_entries WHERE customer_id=$1`, customerID).Scan(&balance); err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(points),0)::bigint FROM loyalty_redemption_holds WHERE customer_id=$1 AND state='HELD' AND expires_at>clock_timestamp()`, customerID).Scan(&reserved); err != nil {
		return LoyaltyHold{}, err
	}
	if balance < reserved || points > balance-reserved {
		return LoyaltyHold{}, ErrLoyaltyInsufficientBalance
	}
	policy, err := activeLoyaltyPolicy(ctx, tx)
	if err != nil {
		return LoyaltyHold{}, err
	}
	discount, err := vndForPoints(points, policy.PointsPerVNDNumerator, policy.PointsPerVNDDenominator)
	if err != nil || discount <= 0 {
		return LoyaltyHold{}, errors.New("active policy does not produce a positive VND discount")
	}
	hold, err := scanLoyaltyHold(tx.QueryRow(ctx, `INSERT INTO loyalty_redemption_holds
		(order_id,customer_id,points,discount_amount_vnd,currency_code,state,expires_at,policy_version)
		VALUES($1,$2,$3,$4,'VND','HELD',clock_timestamp()+($5 * interval '1 minute'),$6)
		RETURNING id::text,order_id,customer_id,points,discount_amount_vnd,currency_code,state,resource_version,expires_at,policy_version`,
		orderID, customerID, points, discount, policy.HoldTTLMinutes, policy.Version))
	if err != nil {
		if uniqueViolation(err) {
			return LoyaltyHold{}, ErrLoyaltyHoldConflict
		}
		return LoyaltyHold{}, err
	}
	hold.CommandResult = "APPLIED"
	if err := storeLoyaltyCommand(ctx, tx, key, "LOCK_REDEMPTION_POINTS", hash, hold); err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoyaltyHold{}, err
	}
	return hold, nil
}

func (r *LoyaltyRepository) FinalizeRedemptionHold(ctx context.Context, orderID, holdID, key string, expectedVersion int64) (LoyaltyHold, error) {
	if orderID == "" || holdID == "" || expectedVersion < 1 || !validLoyaltyIdempotencyKey(key) {
		return LoyaltyHold{}, errors.New("order, hold, positive expected version and valid idempotency key are required")
	}
	hash := loyaltyPayloadHash(struct {
		Action, OrderID, HoldID string
		Version                 int64
	}{"FinalizeRedemptionPoints", orderID, holdID, expectedVersion})
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LoyaltyHold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockLoyaltyCommand(ctx, tx, key); err != nil {
		return LoyaltyHold{}, err
	}
	if hold, found, err := replayLoyaltyCommand(ctx, tx, key, "FINALIZE_REDEMPTION_POINTS", hash); err != nil {
		return LoyaltyHold{}, err
	} else if found {
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return LoyaltyHold{}, err
		}
		return hold, nil
	}
	if err := lockLoyaltyOrder(ctx, tx, orderID); err != nil {
		return LoyaltyHold{}, err
	}
	hold, err := getLoyaltyHold(ctx, tx, orderID, holdID, true)
	if err != nil {
		return LoyaltyHold{}, err
	}
	if hold.Version != expectedVersion || hold.State != "HELD" {
		return LoyaltyHold{}, ErrLoyaltyHoldConflict
	}
	if err := lockLoyaltyCustomer(ctx, tx, hold.CustomerID); err != nil {
		return LoyaltyHold{}, err
	}
	var expired bool
	if err := tx.QueryRow(ctx, `SELECT expires_at<=clock_timestamp() FROM loyalty_redemption_holds WHERE id=$1::uuid`, hold.ID).Scan(&expired); err != nil {
		return LoyaltyHold{}, err
	}
	if expired {
		return LoyaltyHold{}, ErrLoyaltyHoldConflict
	}
	ledgerKey := "loyalty-redemption:" + hold.ID
	debitHash := loyaltyPayloadHash(struct {
		HoldID, CustomerID string
		Points             int64
		PolicyVersion      string
	}{hold.ID, hold.CustomerID, hold.Points, hold.PolicyVersion})
	if _, err := tx.Exec(ctx, `INSERT INTO loyalty_ledger_entries
		(customer_id,entry_type,points_delta,basis_amount_vnd,source_type,source_id,source_decision_id,idempotency_key,request_hash,policy_version,redemption_hold_id)
		VALUES($1,'REDEMPTION_DEBIT',$2,$3,'CHECKOUT_REDEMPTION',$4,$5,$6,$7,$8,$9::uuid)`,
		hold.CustomerID, -hold.Points, hold.DiscountAmount, hold.ID, key, ledgerKey, debitHash, hold.PolicyVersion, hold.ID); err != nil {
		if uniqueViolation(err) {
			return LoyaltyHold{}, ErrLoyaltyHoldConflict
		}
		return LoyaltyHold{}, err
	}
	hold, err = scanLoyaltyHold(tx.QueryRow(ctx, `UPDATE loyalty_redemption_holds SET state='REDEEMED',resource_version=resource_version+1,finalized_at=clock_timestamp(),updated_at=clock_timestamp()
		WHERE id=$1::uuid RETURNING id::text,order_id,customer_id,points,discount_amount_vnd,currency_code,state,resource_version,expires_at,policy_version`, hold.ID))
	if err != nil {
		return LoyaltyHold{}, err
	}
	hold.CommandResult = "APPLIED"
	if err := storeLoyaltyCommand(ctx, tx, key, "FINALIZE_REDEMPTION_POINTS", hash, hold); err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoyaltyHold{}, err
	}
	return hold, nil
}

func (r *LoyaltyRepository) ReleaseRedemptionByOrder(ctx context.Context, orderID, holdID, key string, expectedVersion int64) (LoyaltyHold, error) {
	if orderID == "" || !validLoyaltyIdempotencyKey(key) || (holdID != "" && expectedVersion < 1) {
		return LoyaltyHold{}, errors.New("order and valid idempotency key are required; an identified hold needs a positive version")
	}
	hash := loyaltyPayloadHash(struct {
		Action, OrderID, HoldID string
		Version                 int64
	}{"ReleaseRedemptionByOrder", orderID, holdID, expectedVersion})
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LoyaltyHold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockLoyaltyCommand(ctx, tx, key); err != nil {
		return LoyaltyHold{}, err
	}
	if hold, found, err := replayLoyaltyCommand(ctx, tx, key, "RELEASE_REDEMPTION_POINTS", hash); err != nil {
		return LoyaltyHold{}, err
	} else if found {
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return LoyaltyHold{}, err
		}
		return hold, nil
	}
	if err := lockLoyaltyOrder(ctx, tx, orderID); err != nil {
		return LoyaltyHold{}, err
	}
	hold, err := getLoyaltyHold(ctx, tx, orderID, holdID, true)
	if errors.Is(err, ErrLoyaltyHoldNotFound) {
		if holdID != "" {
			return LoyaltyHold{}, ErrLoyaltyHoldNotFound
		}
		policy, policyErr := activeLoyaltyPolicy(ctx, tx)
		if policyErr != nil {
			return LoyaltyHold{}, policyErr
		}
		hold, err = scanLoyaltyHold(tx.QueryRow(ctx, `INSERT INTO loyalty_redemption_holds
			(order_id,customer_id,points,discount_amount_vnd,currency_code,state,expires_at,policy_version,released_at)
			VALUES($1,'',0,0,'VND','RELEASED',clock_timestamp(),$2,clock_timestamp())
			RETURNING id::text,order_id,customer_id,points,discount_amount_vnd,currency_code,state,resource_version,expires_at,policy_version`, orderID, policy.Version))
		if err != nil {
			return LoyaltyHold{}, err
		}
		hold.Tombstone = true
		hold.CommandResult = "APPLIED"
	} else if err != nil {
		return LoyaltyHold{}, err
	} else {
		if holdID != "" && (hold.ID != holdID || hold.Version != expectedVersion) {
			return LoyaltyHold{}, ErrLoyaltyHoldConflict
		}
		switch hold.State {
		case "HELD":
			if err := lockLoyaltyCustomer(ctx, tx, hold.CustomerID); err != nil {
				return LoyaltyHold{}, err
			}
			hold, err = scanLoyaltyHold(tx.QueryRow(ctx, `UPDATE loyalty_redemption_holds SET state='RELEASED',resource_version=resource_version+1,released_at=clock_timestamp(),updated_at=clock_timestamp()
				WHERE id=$1::uuid RETURNING id::text,order_id,customer_id,points,discount_amount_vnd,currency_code,state,resource_version,expires_at,policy_version`, hold.ID))
			if err != nil {
				return LoyaltyHold{}, err
			}
			hold.CommandResult = "APPLIED"
		case "RELEASED", "EXPIRED":
			hold.CommandResult = "ALREADY_APPLIED"
		case "REDEEMED":
			return LoyaltyHold{}, ErrLoyaltyHoldConflict
		default:
			return LoyaltyHold{}, ErrLoyaltyHoldConflict
		}
	}
	if err := storeLoyaltyCommand(ctx, tx, key, "RELEASE_REDEMPTION_POINTS", hash, hold); err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoyaltyHold{}, err
	}
	return hold, nil
}

func (r *LoyaltyRepository) GetRedemptionByOrder(ctx context.Context, orderID, holdID string) (LoyaltyHold, error) {
	if orderID == "" {
		return LoyaltyHold{}, errors.New("order id is required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return LoyaltyHold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	hold, err := getLoyaltyHold(ctx, tx, orderID, holdID, false)
	if err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&hold.ObservedAt); err != nil {
		return LoyaltyHold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoyaltyHold{}, err
	}
	return hold, nil
}

func (r *LoyaltyRepository) ExpireDueRedemptionHolds(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	tag, err := r.pool.Exec(ctx, `WITH due AS (
		SELECT id FROM loyalty_redemption_holds WHERE state='HELD' AND expires_at<=clock_timestamp() ORDER BY expires_at,id FOR UPDATE SKIP LOCKED LIMIT $1
	) UPDATE loyalty_redemption_holds h SET state='EXPIRED',resource_version=resource_version+1,updated_at=clock_timestamp() FROM due WHERE h.id=due.id`, limit)
	return tag.RowsAffected(), err
}

func getLoyaltyHold(ctx context.Context, tx pgx.Tx, orderID, holdID string, lock bool) (LoyaltyHold, error) {
	sql := `SELECT id::text,order_id,customer_id,points,discount_amount_vnd,currency_code,state,resource_version,expires_at,policy_version FROM loyalty_redemption_holds WHERE order_id=$1`
	args := []any{orderID}
	if holdID != "" {
		sql += ` AND id=$2::uuid`
		args = append(args, holdID)
	}
	if lock {
		sql += ` FOR UPDATE`
	}
	hold, err := scanLoyaltyHold(tx.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return LoyaltyHold{}, ErrLoyaltyHoldNotFound
	}
	return hold, err
}

func scanLoyaltyHold(row interface{ Scan(...any) error }) (LoyaltyHold, error) {
	var hold LoyaltyHold
	err := row.Scan(&hold.ID, &hold.OrderID, &hold.CustomerID, &hold.Points, &hold.DiscountAmount,
		&hold.Currency, &hold.State, &hold.Version, &hold.ExpiresAt, &hold.PolicyVersion)
	if err == nil && hold.Points == 0 && hold.State == "RELEASED" {
		hold.Tombstone = true
	}
	return hold, err
}

func replayLoyaltyCommand(ctx context.Context, tx pgx.Tx, key, commandType, hash string) (LoyaltyHold, bool, error) {
	var storedType, storedHash string
	var response []byte
	err := tx.QueryRow(ctx, `SELECT command_type,request_hash,response_json FROM loyalty_command_idempotency WHERE idempotency_key=$1`, key).Scan(&storedType, &storedHash, &response)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoyaltyHold{}, false, nil
	}
	if err != nil {
		return LoyaltyHold{}, false, err
	}
	if storedType != commandType || storedHash != hash {
		return LoyaltyHold{}, false, ErrLoyaltyIdempotencyConflict
	}
	var hold LoyaltyHold
	if err := json.Unmarshal(response, &hold); err != nil {
		return LoyaltyHold{}, false, err
	}
	return hold, true, nil
}

func storeLoyaltyCommand(ctx context.Context, tx pgx.Tx, key, commandType, hash string, hold LoyaltyHold) error {
	response, err := json.Marshal(hold)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO loyalty_command_idempotency(idempotency_key,command_type,request_hash,resource_id,response_json) VALUES($1,$2,$3,$4::uuid,$5)`, key, commandType, hash, hold.ID, response)
	return err
}

func lockLoyaltyCommand(ctx context.Context, tx pgx.Tx, key string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "loyalty-command:"+key)
	return err
}

func lockLoyaltyOrder(ctx context.Context, tx pgx.Tx, orderID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "loyalty-order:"+orderID)
	return err
}

func lockLoyaltyCustomer(ctx context.Context, tx pgx.Tx, customerID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "loyalty-customer:"+customerID)
	return err
}

func activeLoyaltyPolicy(ctx context.Context, tx pgx.Tx) (LoyaltyPolicy, error) {
	var policy LoyaltyPolicy
	err := tx.QueryRow(ctx, `SELECT policy_version,earn_rate_basis_points,points_per_vnd_numerator,points_per_vnd_denominator,base_tier_code,rounding_mode,hold_ttl_minutes,points_expire_days FROM loyalty_policies WHERE status='ACTIVE'`).Scan(
		&policy.Version, &policy.EarnRateBasisPoints, &policy.PointsPerVNDNumerator, &policy.PointsPerVNDDenominator,
		&policy.BaseTierCode, &policy.RoundingMode, &policy.HoldTTLMinutes, &policy.PointsExpireDays)
	return policy, err
}

func vndForPoints(points, numerator, denominator int64) (int64, error) {
	if points < 0 || numerator <= 0 || denominator <= 0 {
		return 0, errors.New("invalid points conversion rate")
	}
	value := loyaltyBigInt(points, denominator)
	value.Quo(value, loyaltyBigInt(numerator, 1))
	if !value.IsInt64() {
		return 0, errors.New("redemption amount overflow")
	}
	return value.Int64(), nil
}

func loyaltyBigInt(a, b int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
}

func loyaltyPayloadHash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
