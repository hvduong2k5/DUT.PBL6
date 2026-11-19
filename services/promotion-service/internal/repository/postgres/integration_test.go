package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/promotion-service/internal/domain"
	"github.com/omamx/promotion-service/migrations"
)

// Set PROMOTION_TEST_DATABASE_URL to run this test against a disposable PostgreSQL schema.
func TestPromotionPostgresIntegration(t *testing.T) {
	databaseURL := os.Getenv("PROMOTION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set PROMOTION_TEST_DATABASE_URL to run PostgreSQL integration coverage")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	schema := "promotion_test_" + time.Now().UTC().Format("20060102150405") + "_" + randomSchemaSuffix(t)
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupCtx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
	}()

	serviceConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse service test database URL: %v", err)
	}
	serviceConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, serviceConfig)
	if err != nil {
		t.Fatalf("connect service test pool: %v", err)
	}
	defer pool.Close()
	if _, err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	now := time.Now().UTC()
	admin := NewAdminCouponRepository(pool)
	maximum := int64(1)
	coupon, err := admin.Create(ctx, CouponInput{
		Code: "INTEGRATION10", DiscountType: domain.DiscountFixedAmount, DiscountValue: 1000,
		MaxRedemptions: &maximum, StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour),
	}, CouponActor{ID: "integration-test", Role: "SYSTEM"})
	if err != nil {
		t.Fatalf("create coupon: %v", err)
	}
	if coupon.Status != "DRAFT" {
		t.Fatalf("new coupon status = %q, want DRAFT", coupon.Status)
	}
	coupon, err = admin.Publish(ctx, coupon.ID, CouponActor{ID: "integration-test", Role: "SYSTEM"})
	if err != nil {
		t.Fatalf("publish coupon: %v", err)
	}
	if coupon.Status != "ACTIVE" {
		t.Fatalf("published coupon status = %q, want ACTIVE", coupon.Status)
	}

	vouchers := NewVoucherRepository(pool)
	request := LockRequest{IdempotencyKey: "integration-voucher-lock-0001", VoucherCode: "integration10", OrderID: "integration-order-1", CustomerID: "integration-customer-1", Subtotal: 5000, Currency: "VND", TTL: 10 * time.Minute, ContractVersion: "order-3.2"}
	hold, err := vouchers.LockVoucher(ctx, request, now)
	if err != nil {
		t.Fatalf("lock voucher: %v", err)
	}
	if hold.State != "HELD" || hold.DiscountAmount != 1000 {
		t.Fatalf("unexpected voucher hold: %+v", hold)
	}
	replayed, err := vouchers.LockVoucher(ctx, request, now)
	if err != nil || !replayed.Replay || replayed.ID != hold.ID {
		t.Fatalf("voucher retry = %+v, %v; want same hold replay", replayed, err)
	}
	version := hold.Version
	finalized, err := vouchers.FinalizeHold(ctx, "integration-voucher-finalize-0001", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", hold.OrderID, hold.ID, &version, now)
	if err != nil {
		t.Fatalf("finalize voucher hold: %v", err)
	}
	if finalized.State != "FINALIZED" {
		t.Fatalf("finalized state = %q", finalized.State)
	}
	_, _, err = vouchers.CalculateDiscount(ctx, "INTEGRATION10", "another-customer", 5000, 0, "VND", now)
	if !errors.Is(err, domain.ErrUsageLimit) {
		t.Fatalf("discount after exhausted redemption limit error = %v, want usage limit", err)
	}

	loyalty := NewLoyaltyRepository(pool)
	earned, err := loyalty.RecordPurchaseEarn(ctx, "integration-customer-2", "integration-order-2", "completion-decision-2", "loyalty-dev-v1", "integration-loyalty-earn-0001", 10000)
	if err != nil {
		t.Fatalf("record purchase earn: %v", err)
	}
	if earned.PointsDelta != 100 {
		t.Fatalf("earned points = %d, want 100", earned.PointsDelta)
	}
	earnReplay, err := loyalty.RecordPurchaseEarn(ctx, "integration-customer-2", "integration-order-2", "completion-decision-2", "loyalty-dev-v1", "integration-loyalty-earn-0001", 10000)
	if err != nil || !earnReplay.Replay || earnReplay.ID != earned.ID {
		t.Fatalf("earn retry = %+v, %v; want same entry replay", earnReplay, err)
	}

	redemption, err := loyalty.LockRedemptionPoints(ctx, "integration-order-redemption", "integration-customer-2", "integration-redemption-lock-0001", 40)
	if err != nil {
		t.Fatalf("lock redemption points: %v", err)
	}
	if redemption.State != "HELD" || redemption.DiscountAmount != 40 {
		t.Fatalf("unexpected redemption hold: %+v", redemption)
	}
	redemption, err = loyalty.FinalizeRedemptionHold(ctx, redemption.OrderID, redemption.ID, "integration-redemption-finalize-0001", redemption.Version)
	if err != nil {
		t.Fatalf("finalize redemption: %v", err)
	}
	if redemption.State != "REDEEMED" {
		t.Fatalf("redemption state = %q, want REDEEMED", redemption.State)
	}
	page, err := loyalty.GetLedgerPage(ctx, "integration-customer-2", 50, 0)
	if err != nil {
		t.Fatalf("read loyalty ledger: %v", err)
	}
	if page.Balance != 60 {
		t.Fatalf("loyalty balance = %d, want 60", page.Balance)
	}
}

func randomSchemaSuffix(t *testing.T) string {
	t.Helper()
	var value [4]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate test schema suffix: %v", err)
	}
	return hex.EncodeToString(value[:])
}
