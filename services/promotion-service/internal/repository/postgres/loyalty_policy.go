package postgres

import (
	"context"
	"errors"
	"math/big"
)

type LoyaltyPolicy struct {
	Version                 string
	EarnRateBasisPoints     int64
	PointsPerVNDNumerator   int64
	PointsPerVNDDenominator int64
	BaseTierCode            string
	RoundingMode            string
	HoldTTLMinutes          int64
	PointsExpireDays        *int64
}

func (r *LoyaltyRepository) GetActivePolicy(ctx context.Context) (LoyaltyPolicy, error) {
	var policy LoyaltyPolicy
	err := r.pool.QueryRow(ctx, `SELECT policy_version,earn_rate_basis_points,points_per_vnd_numerator,points_per_vnd_denominator,base_tier_code,rounding_mode,hold_ttl_minutes,points_expire_days
		FROM loyalty_policies WHERE status='ACTIVE'`).Scan(&policy.Version, &policy.EarnRateBasisPoints,
		&policy.PointsPerVNDNumerator, &policy.PointsPerVNDDenominator, &policy.BaseTierCode,
		&policy.RoundingMode, &policy.HoldTTLMinutes, &policy.PointsExpireDays)
	return policy, err
}

func (r *LoyaltyRepository) GetPolicy(ctx context.Context, version string) (LoyaltyPolicy, error) {
	var policy LoyaltyPolicy
	err := r.pool.QueryRow(ctx, `SELECT policy_version,earn_rate_basis_points,points_per_vnd_numerator,points_per_vnd_denominator,base_tier_code,rounding_mode,hold_ttl_minutes,points_expire_days
		FROM loyalty_policies WHERE policy_version=$1 AND status IN ('ACTIVE','RETIRED')`, version).Scan(&policy.Version, &policy.EarnRateBasisPoints,
		&policy.PointsPerVNDNumerator, &policy.PointsPerVNDDenominator, &policy.BaseTierCode,
		&policy.RoundingMode, &policy.HoldTTLMinutes, &policy.PointsExpireDays)
	return policy, err
}

func (p LoyaltyPolicy) PointsToVNDIntegerRate() (int64, error) {
	if p.PointsPerVNDDenominator <= 0 || p.PointsPerVNDNumerator%p.PointsPerVNDDenominator != 0 {
		return 0, errors.New("active points-to-VND rate cannot be represented by the frontend integer contract")
	}
	return p.PointsPerVNDNumerator / p.PointsPerVNDDenominator, nil
}

func (p LoyaltyPolicy) CalculatePurchasePoints(eligibleAmountVND int64) (int64, error) {
	if eligibleAmountVND < 0 || p.EarnRateBasisPoints < 0 || p.EarnRateBasisPoints > 10000 {
		return 0, errors.New("invalid eligible amount or earn rate")
	}
	return scaledInteger(eligibleAmountVND, p.EarnRateBasisPoints, 10000)
}

func (p LoyaltyPolicy) CalculateRefundPoints(refundAmountVND int64) (int64, error) {
	if refundAmountVND <= 0 || p.PointsPerVNDNumerator <= 0 || p.PointsPerVNDDenominator <= 0 {
		return 0, errors.New("invalid refund amount or points-to-VND rate")
	}
	return scaledInteger(refundAmountVND, p.PointsPerVNDNumerator, p.PointsPerVNDDenominator)
}

func scaledInteger(value, numerator, denominator int64) (int64, error) {
	if value < 0 || numerator < 0 || denominator <= 0 {
		return 0, errors.New("invalid non-negative integer ratio")
	}
	product := new(big.Int).Mul(big.NewInt(value), big.NewInt(numerator))
	product.Quo(product, big.NewInt(denominator))
	if !product.IsInt64() {
		return 0, errors.New("loyalty point calculation overflow")
	}
	return product.Int64(), nil
}

func (r *LoyaltyRepository) RecordPurchaseEarn(ctx context.Context, customerID, orderID, completionDecisionID, policyVersion, idempotencyKey string, eligibleAmountVND int64) (LoyaltyEntry, error) {
	policy, err := r.GetPolicy(ctx, policyVersion)
	if err != nil {
		return LoyaltyEntry{}, err
	}
	points, err := policy.CalculatePurchasePoints(eligibleAmountVND)
	if err != nil {
		return LoyaltyEntry{}, err
	}
	if points == 0 {
		return LoyaltyEntry{}, ErrNoLoyaltyPointsEarned
	}
	return r.AppendEntry(ctx, LoyaltyEntryInput{
		CustomerID: customerID, EntryType: "PURCHASE_EARN", PointsDelta: points,
		BasisAmountVND: &eligibleAmountVND, SourceType: "ORDER_COMPLETION", SourceID: orderID,
		SourceDecisionID: completionDecisionID, IdempotencyKey: idempotencyKey, PolicyVersion: policy.Version,
	})
}

func (r *LoyaltyRepository) RecordRefundConversion(ctx context.Context, customerID, refundID, refundDecisionID, consentReferenceID, policyVersion, idempotencyKey string, refundAmountVND int64) (LoyaltyEntry, error) {
	policy, err := r.GetPolicy(ctx, policyVersion)
	if err != nil {
		return LoyaltyEntry{}, err
	}
	points, err := policy.CalculateRefundPoints(refundAmountVND)
	if err != nil {
		return LoyaltyEntry{}, err
	}
	return r.AppendEntry(ctx, LoyaltyEntryInput{
		CustomerID: customerID, EntryType: "REFUND_CONVERSION", PointsDelta: points,
		BasisAmountVND: &refundAmountVND, SourceType: "REFUND_DECISION", SourceID: refundID,
		SourceDecisionID: refundDecisionID, ConsentReferenceID: consentReferenceID,
		IdempotencyKey: idempotencyKey, PolicyVersion: policy.Version,
	})
}
