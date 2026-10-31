package persistenceintegration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/migrations"
)

func TestLegacyWorkersAndReadsCannotUseTargetCohort(t *testing.T) {
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("NOT_RUN: real PostgreSQL16 not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "core_fence_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		_, e := admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
		if e != nil {
			t.Error(e)
		}
	}()
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	cipher, _ := persistence.NewCipher("test", map[string][]byte{"test": bytes.Repeat([]byte{1}, 32)})
	store, err := persistence.NewStore(db, cipher, bytes.Repeat([]byte{2}, 32))
	if err != nil {
		t.Fatal(err)
	}
	principal, customer := uuid.NewString(), uuid.NewString()
	input := persistence.AcceptanceInput{Owner: core.Owner{Kind: "CUSTOMER", PrincipalID: principal, CustomerID: &customer}, Method: "VIETQR", Key: uuid.NewString(), RequestHash: strings.Repeat("a", 64),
		QuoteID: uuid.NewString(), CartID: uuid.NewString(), CartRevision: 1, SelectionDigest: core.SelectionHash(strings.Repeat("b", 64)), QuoteHash: core.PayloadHash(strings.Repeat("c", 64)),
		QuoteExpiresAt: time.Now().Add(time.Minute), AcceptedJSON: []byte(`{"synthetic":true}`)}
	a, err := store.Accept(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	legacy := &app.Service{DB: db}
	p := domain.Principal{ID: principal, Role: "CUSTOMER", CustomerID: customer}
	if worked, err := legacy.WorkPlacement(ctx); err != nil || worked {
		t.Fatal("legacy claimed target operation", worked, err)
	}
	if worked, err := legacy.WorkJob(ctx); err != nil || worked {
		t.Fatal("legacy claimed target job", worked, err)
	}
	if _, err = legacy.Operation(ctx, a.OperationID, p); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("legacy operation read", err)
	}
	if _, code, err := legacy.Accept(ctx, p, input.Key, domain.CheckoutInput{QuoteID: input.QuoteID, Revision: 1, Method: "VIETQR"}); err == nil || code != 409 {
		t.Fatal("cross-cohort replay attempted Redis fallback", code, err)
	}
	// Storage projection fixture intentionally uses opaque ciphertext. If the
	// legacy adapter crosses the boundary it can read or try to cancel this row.
	_, err = db.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,customer_id,method,status,stock_status,subtotal,shipping_fee,final_amount,
 snapshot_cipher,reservation_id,payment_expires_at,cohort,owner_kind,snapshot_id,snapshot_schema_version,snapshot_hash,core_state_cipher)
 VALUES($1,$2,$3,$4,$5,$6,'VIETQR','PENDING_PAYMENT','RESERVED',1,0,1,$7,'synthetic-reservation',clock_timestamp()-interval '1 minute',
 'order-core-1','CUSTOMER',$8,'1.0',$9,$7)`, a.PlannedOrderID, a.OperationID, "TARGET-"+a.PlannedOrderID, p.Scope(), principal, customer, []byte("opaque-target-fixture"), uuid.NewString(), strings.Repeat("d", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.GetOrder(ctx, a.PlannedOrderID, p, false); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("legacy order read", err)
	}
	orders, err := legacy.ListOrders(ctx, p, false, false, "", "", 20)
	if err != nil || len(orders) != 0 {
		t.Fatal("legacy list", err)
	}
	if err = legacy.SweepTimeouts(ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	if err = db.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", a.PlannedOrderID).Scan(&status); err != nil || status != "PENDING_PAYMENT" {
		t.Fatal("legacy timeout mutated target", err)
	}
}
