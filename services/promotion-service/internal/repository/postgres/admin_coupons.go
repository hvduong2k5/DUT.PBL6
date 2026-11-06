package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/domain"
)

type CouponInput struct {
	Code                      string
	DiscountType              domain.DiscountType
	DiscountValue             int64
	MaxDiscountAmount         *int64
	MinimumOrderAmount        int64
	MaxRedemptions            *int64
	MaxRedemptionsPerCustomer *int64
	StartsAt                  time.Time
	EndsAt                    time.Time
}

type AdminCoupon struct {
	domain.Coupon
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CouponActor struct {
	ID   string
	Role string
}

type AdminCouponRepository struct{ pool *pgxpool.Pool }

func NewAdminCouponRepository(pool *pgxpool.Pool) *AdminCouponRepository {
	return &AdminCouponRepository{pool: pool}
}

func (r *AdminCouponRepository) Create(ctx context.Context, input CouponInput, actor CouponActor) (AdminCoupon, error) {
	input.Code = domain.NormalizeCode(input.Code)
	if err := validateCouponInput(input); err != nil {
		return AdminCoupon{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdminCoupon{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	coupon, err := scanAdminCoupon(tx.QueryRow(ctx, `INSERT INTO coupons
		(code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'DRAFT')
		RETURNING id::text,code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status,created_at,updated_at`,
		input.Code, input.DiscountType, input.DiscountValue, input.MaxDiscountAmount, input.MinimumOrderAmount,
		input.MaxRedemptions, input.MaxRedemptionsPerCustomer, input.StartsAt, input.EndsAt))
	if err != nil {
		if uniqueViolation(err) {
			return AdminCoupon{}, domain.ErrCouponCodeConflict
		}
		return AdminCoupon{}, err
	}
	if err := writeAdminAudit(ctx, tx, actor, "COUPON_CREATED", coupon.ID, nil, coupon); err != nil {
		return AdminCoupon{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminCoupon{}, err
	}
	return coupon, nil
}

func (r *AdminCouponRepository) UpdateDraft(ctx context.Context, id string, input CouponInput, actor CouponActor) (AdminCoupon, error) {
	input.Code = domain.NormalizeCode(input.Code)
	if err := validateCouponInput(input); err != nil {
		return AdminCoupon{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdminCoupon{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanAdminCoupon(tx.QueryRow(ctx, `SELECT id::text,code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status,created_at,updated_at FROM coupons WHERE id=$1::uuid FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminCoupon{}, domain.ErrVoucherNotFound
	}
	if err != nil {
		return AdminCoupon{}, err
	}
	if before.Status != "DRAFT" {
		return AdminCoupon{}, domain.ErrCouponImmutable
	}
	after, err := scanAdminCoupon(tx.QueryRow(ctx, `UPDATE coupons SET code=$2,discount_type=$3,discount_value=$4,max_discount_amount=$5,minimum_order_amount=$6,max_redemptions=$7,max_redemptions_per_customer=$8,starts_at=$9,ends_at=$10,updated_at=now()
		WHERE id=$1::uuid AND status='DRAFT'
		RETURNING id::text,code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status,created_at,updated_at`,
		id, input.Code, input.DiscountType, input.DiscountValue, input.MaxDiscountAmount, input.MinimumOrderAmount,
		input.MaxRedemptions, input.MaxRedemptionsPerCustomer, input.StartsAt, input.EndsAt))
	if err != nil {
		if uniqueViolation(err) {
			return AdminCoupon{}, domain.ErrCouponCodeConflict
		}
		return AdminCoupon{}, err
	}
	if err := writeAdminAudit(ctx, tx, actor, "COUPON_DRAFT_UPDATED", id, before, after); err != nil {
		return AdminCoupon{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminCoupon{}, err
	}
	return after, nil
}

func (r *AdminCouponRepository) List(ctx context.Context, limit, offset int) ([]AdminCoupon, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text,code,discount_type,discount_value,max_discount_amount,minimum_order_amount,max_redemptions,max_redemptions_per_customer,starts_at,ends_at,status,created_at,updated_at
		FROM coupons ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminCoupon, 0)
	for rows.Next() {
		coupon, err := scanAdminCoupon(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, coupon)
	}
	return result, rows.Err()
}

func scanAdminCoupon(row interface{ Scan(...any) error }) (AdminCoupon, error) {
	var coupon AdminCoupon
	var discountType string
	err := row.Scan(&coupon.ID, &coupon.Code, &discountType, &coupon.DiscountValue, &coupon.MaxDiscountAmount,
		&coupon.MinimumOrderAmount, &coupon.MaxRedemptions, &coupon.MaxRedemptionsPerCustomer,
		&coupon.StartsAt, &coupon.EndsAt, &coupon.Status, &coupon.CreatedAt, &coupon.UpdatedAt)
	coupon.DiscountType = domain.DiscountType(discountType)
	return coupon, err
}

func validateCouponInput(input CouponInput) error {
	if len(input.Code) < 3 || len(input.Code) > 64 || input.StartsAt.IsZero() || input.EndsAt.IsZero() || !input.EndsAt.After(input.StartsAt) || input.MinimumOrderAmount < 0 {
		return domain.ErrCouponConfigInvalid
	}
	for _, character := range input.Code {
		if !((character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' || character == '-') {
			return domain.ErrCouponConfigInvalid
		}
	}
	switch input.DiscountType {
	case domain.DiscountFixedAmount:
		if input.DiscountValue <= 0 {
			return domain.ErrCouponConfigInvalid
		}
	case domain.DiscountPercentage:
		if input.DiscountValue < 1 || input.DiscountValue > 100 {
			return domain.ErrCouponConfigInvalid
		}
	case domain.DiscountFreeShipping:
		if input.DiscountValue != 0 {
			return domain.ErrCouponConfigInvalid
		}
	default:
		return domain.ErrCouponConfigInvalid
	}
	if input.MaxDiscountAmount != nil && *input.MaxDiscountAmount <= 0 {
		return domain.ErrCouponConfigInvalid
	}
	if input.MaxRedemptions != nil && *input.MaxRedemptions <= 0 || input.MaxRedemptionsPerCustomer != nil && *input.MaxRedemptionsPerCustomer <= 0 {
		return domain.ErrCouponConfigInvalid
	}
	if input.MaxRedemptions != nil && input.MaxRedemptionsPerCustomer != nil && *input.MaxRedemptionsPerCustomer > *input.MaxRedemptions {
		return domain.ErrCouponConfigInvalid
	}
	return nil
}

func writeAdminAudit(ctx context.Context, tx pgx.Tx, actor CouponActor, action, id string, before, after any) error {
	var beforeJSON, afterJSON []byte
	var err error
	if before != nil {
		beforeJSON, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	if after != nil {
		afterJSON, err = json.Marshal(after)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO promotion_admin_audit(actor_id,actor_role,action,resource_type,resource_id,before_state,after_state) VALUES($1,$2,$3,'COUPON',$4,$5,$6)`, strings.TrimSpace(actor.ID), strings.ToUpper(actor.Role), action, id, beforeJSON, afterJSON)
	return err
}

func uniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
