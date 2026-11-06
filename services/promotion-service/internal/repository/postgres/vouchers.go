package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/domain"
)

type VoucherRepository struct{ pool *pgxpool.Pool }

func (r *VoucherRepository) Pool() *pgxpool.Pool { return r.pool }

func NewVoucherRepository(pool *pgxpool.Pool) *VoucherRepository {
	return &VoucherRepository{pool: pool}
}

type Hold struct {
	ID             string
	OrderID        string
	CustomerID     string
	CouponCode     string
	DiscountType   domain.DiscountType
	DiscountAmount int64
	Currency       string
	State          string
	ExpiresAt      time.Time
	Version        int64
	RequestHash    string
	ObservedAt     time.Time
	Replay         bool
	Tombstone      bool
	CommandResult  string
	ReasonCode     string
	DecisionAt     time.Time
	ResultingState string
	DecisionID     string
}

type HoldCommandEvidence struct {
	IdempotencyKey string
	Action         string
	RequestHash    string
	Result         string
	ReasonCode     string
	DecisionID     string
	DecisionAt     time.Time
	ResultingState string
}

type LockRequest struct {
	IdempotencyKey  string
	RequestHash     string
	VoucherCode     string
	OrderID         string
	CustomerID      string
	Subtotal        int64
	ShippingFee     int64
	Currency        string
	TTL             time.Duration
	ContractVersion string
}

func (r *VoucherRepository) CalculateDiscount(ctx context.Context, code, customerID string, subtotal, shippingFee int64, currency string, now time.Time) (domain.Coupon, int64, error) {
	coupon, err := r.findCoupon(ctx, nil, code, false)
	if err != nil {
		return domain.Coupon{}, 0, err
	}
	discount, err := coupon.Discount(subtotal, shippingFee, currency, now)
	if err != nil {
		return coupon, 0, err
	}
	if err := r.checkAvailability(ctx, nil, coupon, customerID, now); err != nil {
		return coupon, 0, err
	}
	return coupon, discount, nil
}

func (r *VoucherRepository) LockVoucher(ctx context.Context, request LockRequest, now time.Time) (Hold, error) {
	if request.IdempotencyKey == "" || request.OrderID == "" || request.CustomerID == "" || domain.NormalizeCode(request.VoucherCode) == "" {
		return Hold{}, errors.New("idempotency key, voucher code, order id and customer id are required")
	}
	if request.TTL <= 0 {
		request.TTL = 15 * time.Minute
	}
	if request.TTL > 15*time.Minute {
		return Hold{}, errors.New("voucher hold TTL cannot exceed 15 minutes")
	}
	request.VoucherCode = domain.NormalizeCode(request.VoucherCode)
	if request.Currency == "" {
		request.Currency = "VND"
	}
	if request.ContractVersion == "" {
		request.ContractVersion = "1"
	}
	if request.RequestHash == "" {
		fingerprint, err := json.Marshal(struct {
			Code, OrderID, CustomerID, Currency string
			Subtotal, ShippingFee               int64
			TTL                                 int64
		}{request.VoucherCode, request.OrderID, request.CustomerID, request.Currency, request.Subtotal, request.ShippingFee, int64(request.TTL)})
		if err != nil {
			return Hold{}, err
		}
		sum := sha256.Sum256(fingerprint)
		request.RequestHash = hex.EncodeToString(sum[:])
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Hold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, request.IdempotencyKey); err != nil {
		return Hold{}, err
	}
	var storedHash, commandType string
	var storedResponse []byte
	err = tx.QueryRow(ctx, `SELECT request_hash,command_type,response_json FROM command_idempotency WHERE idempotency_key=$1`, request.IdempotencyKey).Scan(&storedHash, &commandType, &storedResponse)
	if err == nil {
		if storedHash != request.RequestHash || commandType != "LOCK_VOUCHER" {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		var hold Hold
		if err := json.Unmarshal(storedResponse, &hold); err != nil {
			return Hold{}, err
		}
		if hold.OrderID != request.OrderID {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, err
		}
		return hold, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, err
	}
	if existing, err := scanHold(tx.QueryRow(ctx, `SELECT h.id::text,h.order_id,h.customer_id,c.code,c.discount_type,h.discount_amount,h.currency_code,h.state,h.expires_at,h.resource_version,h.created_at,h.request_hash
		FROM voucher_holds h JOIN coupons c ON c.id=h.coupon_id WHERE h.idempotency_key=$1`, request.IdempotencyKey)); err == nil {
		if existing.RequestHash != request.RequestHash {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		existing.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, err
		}
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, err
	}
	coupon, err := r.findCoupon(ctx, tx, request.VoucherCode, true)
	if err != nil {
		if isVoucherEligibilityError(err) {
			return recordLockRejection(ctx, tx, request, err)
		}
		return Hold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	var orderHasHold bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voucher_holds WHERE order_id=$1)`, request.OrderID).Scan(&orderHasHold); err != nil {
		return Hold{}, err
	}
	if orderHasHold {
		return recordLockRejection(ctx, tx, request, domain.ErrVoucherAlreadyLocked)
	}
	discount, err := coupon.Discount(request.Subtotal, request.ShippingFee, request.Currency, now)
	if err != nil {
		if isVoucherEligibilityError(err) {
			return recordLockRejection(ctx, tx, request, err)
		}
		return Hold{}, err
	}
	if err := r.checkAvailability(ctx, tx, coupon, request.CustomerID, now); err != nil {
		if isVoucherEligibilityError(err) {
			return recordLockRejection(ctx, tx, request, err)
		}
		return Hold{}, err
	}
	expiresAt := now.Add(request.TTL)
	var holdID string
	var version int64
	err = tx.QueryRow(ctx, `INSERT INTO voucher_holds
		(coupon_id,order_id,customer_id,idempotency_key,request_hash,contract_version,discount_amount,currency_code,state,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'HELD',$9) RETURNING id::text,resource_version,created_at`,
		coupon.ID, request.OrderID, request.CustomerID, request.IdempotencyKey, request.RequestHash,
		request.ContractVersion, discount, request.Currency, expiresAt).Scan(&holdID, &version, &now)
	if err != nil {
		if uniqueViolation(err) {
			return Hold{}, domain.ErrVoucherAlreadyLocked
		}
		return Hold{}, err
	}
	hold := Hold{ID: holdID, OrderID: request.OrderID, CustomerID: request.CustomerID, CouponCode: coupon.Code,
		DiscountType: coupon.DiscountType, DiscountAmount: discount, Currency: request.Currency, State: "HELD",
		ExpiresAt: expiresAt, Version: version, RequestHash: request.RequestHash, ObservedAt: now,
		CommandResult: "APPLIED", DecisionAt: now, ResultingState: "HELD", DecisionID: holdID}
	response, err := json.Marshal(hold)
	if err != nil {
		return Hold{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO command_idempotency(idempotency_key,command_type,request_hash,resource_id,response_json,expires_at)
		VALUES($1,'LOCK_VOUCHER',$2,$3::uuid,$4,clock_timestamp()+interval '7 days')`, request.IdempotencyKey, request.RequestHash, hold.ID, response); err != nil {
		return Hold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Hold{}, err
	}
	return hold, nil
}

func recordLockRejection(ctx context.Context, tx pgx.Tx, request LockRequest, reason error) (Hold, error) {
	var decisionAt time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&decisionAt); err != nil {
		return Hold{}, err
	}
	hold := Hold{OrderID: request.OrderID, Currency: request.Currency, ObservedAt: decisionAt,
		DecisionAt: decisionAt, DecisionID: request.IdempotencyKey, CommandResult: "REJECTED",
		ReasonCode: voucherReasonCode(reason)}
	response, err := json.Marshal(hold)
	if err != nil {
		return Hold{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO command_idempotency(idempotency_key,command_type,request_hash,response_json,expires_at)
		VALUES($1,'LOCK_VOUCHER',$2,$3,clock_timestamp()+interval '7 days')`, request.IdempotencyKey, request.RequestHash, response); err != nil {
		return Hold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Hold{}, err
	}
	return hold, nil
}

func isVoucherEligibilityError(err error) bool {
	return errors.Is(err, domain.ErrVoucherNotFound) || errors.Is(err, domain.ErrVoucherInactive) ||
		errors.Is(err, domain.ErrVoucherExpired) || errors.Is(err, domain.ErrMinimumNotMet) ||
		errors.Is(err, domain.ErrUsageLimit) || errors.Is(err, domain.ErrVoucherAlreadyLocked)
}

func voucherReasonCode(err error) string {
	switch {
	case errors.Is(err, domain.ErrVoucherNotFound):
		return "VOUCHER_NOT_FOUND"
	case errors.Is(err, domain.ErrVoucherInactive):
		return "VOUCHER_INACTIVE"
	case errors.Is(err, domain.ErrVoucherExpired):
		return "VOUCHER_EXPIRED"
	case errors.Is(err, domain.ErrMinimumNotMet):
		return "MINIMUM_ORDER_AMOUNT_NOT_MET"
	case errors.Is(err, domain.ErrUsageLimit):
		return "USAGE_LIMIT_EXCEEDED"
	case errors.Is(err, domain.ErrVoucherAlreadyLocked):
		return "VOUCHER_ALREADY_LOCKED"
	default:
		return "VOUCHER_NOT_ELIGIBLE"
	}
}

type holdRow interface{ Scan(...any) error }

func scanHold(row holdRow) (Hold, error) {
	var hold Hold
	var discountType string
	err := row.Scan(&hold.ID, &hold.OrderID, &hold.CustomerID, &hold.CouponCode, &discountType,
		&hold.DiscountAmount, &hold.Currency, &hold.State, &hold.ExpiresAt, &hold.Version, &hold.ObservedAt, &hold.RequestHash)
	hold.DiscountType = domain.DiscountType(discountType)
	return hold, err
}

func (r *VoucherRepository) findCoupon(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string, lock bool) (domain.Coupon, error) {
	if queryer == nil {
		queryer = r.pool
	}
	sql := `SELECT id::text,code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status FROM coupons WHERE code=$1`
	if lock {
		sql += ` FOR UPDATE`
	}
	var coupon domain.Coupon
	var discountType string
	err := queryer.QueryRow(ctx, sql, domain.NormalizeCode(code)).Scan(&coupon.ID, &coupon.Code, &discountType,
		&coupon.DiscountValue, &coupon.MaxDiscountAmount, &coupon.MinimumOrderAmount,
		&coupon.MaxRedemptions, &coupon.MaxRedemptionsPerCustomer, &coupon.StartsAt, &coupon.EndsAt, &coupon.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Coupon{}, domain.ErrVoucherNotFound
	}
	coupon.DiscountType = domain.DiscountType(discountType)
	return coupon, err
}

func (r *VoucherRepository) checkAvailability(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, coupon domain.Coupon, customerID string, now time.Time) error {
	if queryer == nil {
		queryer = r.pool
	}
	var total, customer int64
	err := queryer.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM voucher_usage WHERE coupon_id=$1) +
		(SELECT count(*) FROM voucher_holds WHERE coupon_id=$1 AND state='HELD' AND expires_at>$2),
		(SELECT count(*) FROM voucher_usage WHERE coupon_id=$1 AND customer_id=$3) +
		(SELECT count(*) FROM voucher_holds WHERE coupon_id=$1 AND customer_id=$3 AND state='HELD' AND expires_at>$2)`,
		coupon.ID, now, customerID).Scan(&total, &customer)
	if err != nil {
		return err
	}
	if coupon.MaxRedemptions != nil && total >= *coupon.MaxRedemptions {
		return domain.ErrUsageLimit
	}
	if coupon.MaxRedemptionsPerCustomer != nil && customer >= *coupon.MaxRedemptionsPerCustomer {
		return domain.ErrUsageLimit
	}
	return nil
}

func (r *VoucherRepository) ReleaseHold(ctx context.Context, idempotencyKey, requestHash, orderID, holdID, voucherCode, reason, contractVersion string, expectedVersion *int64, now time.Time) (Hold, error) {
	if idempotencyKey == "" || orderID == "" || reason == "" {
		return Hold{}, fmt.Errorf("idempotency key, order id and release reason are required")
	}
	if requestHash == "" {
		payload, _ := json.Marshal(struct{ OrderID, HoldID, VoucherCode, Reason string }{orderID, holdID, domain.NormalizeCode(voucherCode), reason})
		sum := sha256.Sum256(payload)
		requestHash = hex.EncodeToString(sum[:])
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Hold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, idempotencyKey); err != nil {
		return Hold{}, err
	}
	var storedHash, resourceID, commandType string
	var responseJSON []byte
	err = tx.QueryRow(ctx, `SELECT request_hash,resource_id::text,command_type,response_json FROM command_idempotency WHERE idempotency_key=$1`, idempotencyKey).Scan(&storedHash, &resourceID, &commandType, &responseJSON)
	if err == nil {
		if storedHash != requestHash || commandType != "RELEASE_VOUCHER" {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		var hold Hold
		if err := json.Unmarshal(responseJSON, &hold); err != nil {
			return Hold{}, err
		}
		if hold.OrderID != orderID || (holdID != "" && hold.ID != holdID) {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, err
		}
		return hold, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, err
	}
	hold, err := r.getHold(ctx, tx, holdID, orderID, true)
	createdTombstone := false
	if errors.Is(err, domain.ErrHoldNotFound) && expectedVersion != nil && holdID != "" {
		if *expectedVersion != 1 {
			return Hold{}, fmt.Errorf("%w: expected %d, observed 1", domain.ErrHoldVersionConflict, *expectedVersion)
		}
		if contractVersion == "" {
			contractVersion = "order-3.2"
		}
		hold = Hold{ID: holdID, OrderID: orderID, State: "RELEASED", Version: 1, Currency: "VND", Tombstone: true}
		_, err = tx.Exec(ctx, `INSERT INTO voucher_holds
			(id,coupon_id,order_id,customer_id,idempotency_key,request_hash,contract_version,discount_amount,currency_code,state,resource_version,expires_at,released_at,release_reason,updated_at)
			VALUES($1::uuid,NULL,$2,'','release:'||$3,$4,$5,0,'VND','RELEASED',1,$6,$6,$7,$6)`, holdID, orderID, idempotencyKey, requestHash, contractVersion, now, reason)
		if err != nil {
			if uniqueViolation(err) {
				return Hold{}, domain.ErrVoucherAlreadyLocked
			}
			return Hold{}, err
		}
		createdTombstone = true
	} else if err != nil {
		return Hold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	if voucherCode != "" && domain.NormalizeCode(voucherCode) != hold.CouponCode {
		return Hold{}, domain.ErrHoldNotFound
	}
	if expectedVersion != nil && hold.Version != *expectedVersion {
		return Hold{}, fmt.Errorf("%w: expected %d, observed %d", domain.ErrHoldVersionConflict, *expectedVersion, hold.Version)
	}
	switch hold.State {
	case "HELD":
		if !now.Before(hold.ExpiresAt) {
			hold.State = "EXPIRED"
			hold.CommandResult = "REJECTED"
			hold.ReasonCode = "VOUCHER_HOLD_EXPIRED"
		} else {
			hold.State = "RELEASED"
			hold.CommandResult = "APPLIED"
			hold.ReasonCode = ""
		}
		hold.Version++
		_, err = tx.Exec(ctx, `UPDATE voucher_holds SET state=$2,resource_version=$3,released_at=CASE WHEN $2='RELEASED' THEN $4 ELSE released_at END,release_reason=$5,updated_at=$4 WHERE id=$1::uuid`, hold.ID, hold.State, hold.Version, now, reason)
	case "RELEASED":
		if createdTombstone {
			hold.CommandResult = "APPLIED"
		} else {
			hold.Replay = true
			hold.CommandResult = "ALREADY_APPLIED"
		}
	case "EXPIRED":
		hold.CommandResult = "REJECTED"
		hold.ReasonCode = "VOUCHER_HOLD_EXPIRED"
	case "FINALIZED":
		if contractVersion == "order-3.2" {
			hold.CommandResult = "REJECTED"
			hold.ReasonCode = "VOUCHER_HOLD_FINALIZED"
		} else {
			return Hold{}, domain.ErrHoldTransition
		}
	default:
		return Hold{}, domain.ErrHoldTransition
	}
	if err != nil {
		return Hold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&hold.ObservedAt); err != nil {
		return Hold{}, err
	}
	if hold.DecisionAt.IsZero() {
		hold.DecisionAt = hold.ObservedAt
	}
	hold.ResultingState = hold.State
	response, err := json.Marshal(hold)
	if err != nil {
		return Hold{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO command_idempotency(idempotency_key,command_type,request_hash,resource_id,response_json,expires_at) VALUES($1,'RELEASE_VOUCHER',$2,$3::uuid,$4,now()+interval '7 days')`, idempotencyKey, requestHash, hold.ID, response)
	if err != nil {
		return Hold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Hold{}, err
	}
	return hold, nil
}

func (r *VoucherRepository) FinalizeHold(ctx context.Context, idempotencyKey, requestHash, orderID, holdID string, expectedVersion *int64, now time.Time) (Hold, error) {
	if idempotencyKey == "" || orderID == "" {
		return Hold{}, errors.New("idempotency key and order id are required")
	}
	if requestHash == "" {
		payload, _ := json.Marshal(struct{ OrderID, HoldID string }{orderID, holdID})
		sum := sha256.Sum256(payload)
		requestHash = hex.EncodeToString(sum[:])
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Hold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, idempotencyKey); err != nil {
		return Hold{}, err
	}
	var storedHash, resourceID, commandType string
	var responseJSON []byte
	err = tx.QueryRow(ctx, `SELECT request_hash,resource_id::text,command_type,response_json FROM command_idempotency WHERE idempotency_key=$1`, idempotencyKey).Scan(&storedHash, &resourceID, &commandType, &responseJSON)
	if err == nil {
		if storedHash != requestHash || commandType != "FINALIZE_VOUCHER_HOLD" {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		var hold Hold
		if err := json.Unmarshal(responseJSON, &hold); err != nil {
			return Hold{}, err
		}
		if hold.OrderID != orderID || (holdID != "" && hold.ID != holdID) {
			return Hold{}, domain.ErrIdempotencyConflict
		}
		hold.Replay = true
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, err
		}
		return hold, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, err
	}
	hold, err := r.getHold(ctx, tx, holdID, orderID, true)
	if err != nil {
		return Hold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return Hold{}, err
	}
	if expectedVersion != nil && hold.Version != *expectedVersion {
		return Hold{}, fmt.Errorf("%w: expected %d, observed %d", domain.ErrHoldVersionConflict, *expectedVersion, hold.Version)
	}
	switch hold.State {
	case "HELD":
		if !now.Before(hold.ExpiresAt) {
			hold.State = "EXPIRED"
			hold.Version++
			hold.CommandResult = "REJECTED"
			hold.ReasonCode = "VOUCHER_HOLD_EXPIRED"
			_, err = tx.Exec(ctx, `UPDATE voucher_holds SET state='EXPIRED',resource_version=$2,updated_at=$3 WHERE id=$1::uuid`, hold.ID, hold.Version, now)
		} else {
			hold.State = "FINALIZED"
			hold.Version++
			hold.CommandResult = "APPLIED"
			hold.ReasonCode = ""
			_, err = tx.Exec(ctx, `UPDATE voucher_holds SET state='FINALIZED',resource_version=$2,finalized_at=$3,updated_at=$3 WHERE id=$1::uuid`, hold.ID, hold.Version, now)
			if err == nil {
				_, err = tx.Exec(ctx, `INSERT INTO voucher_usage(hold_id,coupon_id,order_id,customer_id,discount_amount,finalized_at)
					SELECT id,coupon_id,order_id,customer_id,discount_amount,$2 FROM voucher_holds WHERE id=$1::uuid`, hold.ID, now)
			}
		}
	case "FINALIZED":
		// Finalization is idempotent even when it is retried with a new command key.
		hold.Replay = true
		hold.CommandResult = "ALREADY_APPLIED"
		hold.ReasonCode = ""
	case "RELEASED", "EXPIRED":
		hold.CommandResult = "REJECTED"
		if hold.State == "EXPIRED" {
			hold.ReasonCode = "VOUCHER_HOLD_EXPIRED"
		} else {
			hold.ReasonCode = "VOUCHER_HOLD_RELEASED"
		}
	default:
		return Hold{}, domain.ErrHoldTransition
	}
	if err != nil {
		return Hold{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&hold.ObservedAt); err != nil {
		return Hold{}, err
	}
	if hold.DecisionAt.IsZero() {
		hold.DecisionAt = hold.ObservedAt
	}
	hold.ResultingState = hold.State
	response, err := json.Marshal(hold)
	if err != nil {
		return Hold{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO command_idempotency(idempotency_key,command_type,request_hash,resource_id,response_json,expires_at) VALUES($1,'FINALIZE_VOUCHER_HOLD',$2,$3::uuid,$4,now()+interval '7 days')`, idempotencyKey, requestHash, hold.ID, response)
	if err != nil {
		return Hold{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Hold{}, err
	}
	return hold, nil
}

func (r *VoucherRepository) GetHoldByOrder(ctx context.Context, orderID, holdID, queryAction, idempotencyKey, requestHash string) (Hold, *HoldCommandEvidence, error) {
	if orderID == "" {
		return Hold{}, nil, errors.New("order id is required")
	}
	if queryAction != "" && (idempotencyKey == "" || requestHash == "") {
		return Hold{}, nil, errors.New("command evidence filter requires action, idempotency key and request hash")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Hold{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	hold, err := r.getHold(ctx, tx, holdID, orderID, false)
	if err != nil {
		return Hold{}, nil, err
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&hold.ObservedAt); err != nil {
		return Hold{}, nil, err
	}
	var proof *HoldCommandEvidence
	if queryAction != "" {
		proof, err = r.findHoldCommandEvidence(ctx, tx, hold, queryAction, idempotencyKey, requestHash)
		if err != nil {
			return Hold{}, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Hold{}, nil, err
	}
	return hold, proof, nil
}

func (r *VoucherRepository) findHoldCommandEvidence(ctx context.Context, tx pgx.Tx, hold Hold, action, key, requestHash string) (*HoldCommandEvidence, error) {
	proof := &HoldCommandEvidence{IdempotencyKey: key, Action: action, RequestHash: requestHash, DecisionID: hold.ID}
	switch action {
	case "ValidateAndLockVoucher":
		var storedHash string
		var decidedAt time.Time
		err := tx.QueryRow(ctx, `SELECT request_hash,created_at FROM voucher_holds WHERE idempotency_key=$1 AND order_id=$2 AND id=$3::uuid`, key, hold.OrderID, hold.ID).Scan(&storedHash, &decidedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		if err != nil {
			return nil, err
		}
		if storedHash != requestHash {
			return nil, domain.ErrIdempotencyConflict
		}
		proof.Result, proof.DecisionAt, proof.ResultingState = "APPLIED", decidedAt, "HELD"
	case "FinalizeVoucherHold", "ReleaseVoucherHoldByOrder":
		expectedType := "FINALIZE_VOUCHER_HOLD"
		if action == "ReleaseVoucherHoldByOrder" {
			expectedType = "RELEASE_VOUCHER"
		}
		var storedHash, resourceID string
		var responseJSON []byte
		err := tx.QueryRow(ctx, `SELECT request_hash,resource_id::text,response_json FROM command_idempotency WHERE idempotency_key=$1 AND command_type=$2`, key, expectedType).Scan(&storedHash, &resourceID, &responseJSON)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		if err != nil {
			return nil, err
		}
		if storedHash != requestHash || resourceID != hold.ID {
			return nil, domain.ErrIdempotencyConflict
		}
		var original Hold
		if err := json.Unmarshal(responseJSON, &original); err != nil {
			return nil, err
		}
		proof.DecisionAt = original.DecisionAt
		if proof.DecisionAt.IsZero() {
			proof.DecisionAt = original.ObservedAt
		}
		proof.ResultingState = original.ResultingState
		if proof.ResultingState == "" {
			proof.ResultingState = original.State
		}
		proof.ReasonCode = original.ReasonCode
		proof.Result = original.CommandResult
		if proof.Result == "" {
			proof.Result = "APPLIED"
		}
		if original.Replay && proof.Result != "REJECTED" {
			proof.Result = "ALREADY_APPLIED"
		}
	default:
		return nil, errors.New("query_action must name a supported voucher command")
	}
	return proof, nil
}

func (r *VoucherRepository) getHold(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, holdID, orderID string, lock bool) (Hold, error) {
	sql := `SELECT h.id::text,h.order_id,h.customer_id,COALESCE(c.code,''),COALESCE(c.discount_type,''),h.discount_amount,h.currency_code,h.state,h.expires_at,h.resource_version,(h.coupon_id IS NULL) FROM voucher_holds h LEFT JOIN coupons c ON c.id=h.coupon_id WHERE h.order_id=$1`
	args := []any{orderID}
	if holdID != "" {
		sql += ` AND h.id=$2::uuid`
		args = append(args, holdID)
	}
	if lock {
		sql += ` FOR UPDATE OF h`
	}
	var hold Hold
	var discountType string
	err := queryer.QueryRow(ctx, sql, args...).Scan(&hold.ID, &hold.OrderID, &hold.CustomerID, &hold.CouponCode,
		&discountType, &hold.DiscountAmount, &hold.Currency, &hold.State, &hold.ExpiresAt, &hold.Version, &hold.Tombstone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, domain.ErrHoldNotFound
	}
	hold.DiscountType = domain.DiscountType(discountType)
	return hold, err
}
