package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/migrations"
)

func TestCustomerTraceabilityOwnershipAndPeerIsolationPostgres(t *testing.T) {
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "trace_read_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	block, _ := aes.NewCipher(bytes.Repeat([]byte{29}, 32))
	aead, _ := cipher.NewGCM(block)
	service := &Service{DB: db, AEAD: aead}
	actor := domain.Principal{Role: "CUSTOMER", ID: uuid.NewString()}
	order, operation := uuid.NewString(), uuid.NewString()
	sealed, err := service.Seal(domain.Quote{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash) VALUES($1,$2,$3,$4,'SUCCEEDED','input','hash')`, operation, actor.Scope(), actor.ID, order); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,warehouse_id,method,status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,provider_reference,payment_expires_at) VALUES($1::uuid,$2,$1::text,$3,$4,'WH-A','COD','CONFIRMED_COD','COMMITTED',200,0,200,$5,'reservation','trace-read-test',clock_timestamp()+interval '1 day')`, order, operation, actor.Scope(), actor.ID, sealed); err != nil {
		t.Fatal(err)
	}
	assertError := func(id string, p domain.Principal, code string) {
		t.Helper()
		_, err := service.CustomerTraceability(ctx, id, p)
		if problem, ok := err.(*domain.Error); !ok || problem.Code != code {
			t.Fatal("unexpected error", err)
		}
	}
	assertError(order, domain.Principal{Role: "CUSTOMER", ID: uuid.NewString()}, "PERMISSION_DENIED")
	assertError(order, domain.Principal{Role: "ADMIN", ID: actor.ID}, "CUSTOMER_REQUIRED")
	assertError(uuid.NewString(), actor, "ORDER_NOT_FOUND")
	assertError("not-a-uuid", actor, "ORDER_NOT_FOUND")
	result, err := service.CustomerTraceability(ctx, order, actor)
	if err != nil || result.State != "PACKING_PENDING" || result.Items == nil || result.Total != 0 {
		t.Fatal(result, err)
	}
	if _, err = db.Exec(ctx, `UPDATE orders SET status='CANCELLED_BY_USER' WHERE id=$1`, order); err != nil {
		t.Fatal(err)
	}
	result, err = service.CustomerTraceability(ctx, order, actor)
	if err != nil || result.State != "NOT_APPLICABLE" {
		t.Fatal(result, err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO fulfillment_shipping_handoffs(order_id,package_id,task_id,sealed_version,request_hash,request_cipher) VALUES($1,$2,$3,7,'hash','cipher')`, order, uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	// Old snapshots or unavailable providers must never invent sealed lines.
	result, err = service.CustomerTraceability(ctx, order, actor)
	if err != nil || result.State != "UNAVAILABLE" || len(result.Items) != 0 {
		t.Fatal(result, err)
	}
	var version int64
	var status string
	if err = db.QueryRow(ctx, `SELECT version,status FROM orders WHERE id=$1`, order).Scan(&version, &status); err != nil {
		t.Fatal(err)
	}
	if version != 1 || status != "CANCELLED_BY_USER" {
		t.Fatal("trace read changed lifecycle", version, status)
	}
}
