package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/migrations"
)

func TestAdminProjectionScopeCursorAndSensitiveRead(t *testing.T) {
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("explicit isolated PostgreSQL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "admin_read_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(bytes.Repeat([]byte{23}, 32))
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{DB: db, AEAD: aead, C: Config{JWTSecret: strings.Repeat("test-secret-", 4)}}
	actorID := uuid.NewString()
	reader := func(view, sensitive string) AdminOrderReader {
		grants := []domain.PermissionGrant{{PermissionCode: "ORDER_VIEW", ScopeType: "WAREHOUSE", ScopeID: view}}
		if sensitive != "" {
			grants = append(grants, domain.PermissionGrant{PermissionCode: "ORDER_SENSITIVE_VIEW", ScopeType: "WAREHOUSE", ScopeID: sensitive})
		}
		r, e := NewAdminOrderReader(domain.Principal{ID: actorID, Role: "EMPLOYEE", IAM: true, Grants: grants})
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	quote := domain.Quote{}
	quote.Address.Name = "Snapshot recipient"
	quote.Address.Phone = "0901234567"
	quote.Address.Street = "Frozen street"
	sealed, err := s.Seal(quote)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, id := range ids {
		op := uuid.NewString()
		warehouse := "WH-A"
		if i == 0 {
			warehouse = "WH-B"
		}
		if _, err = db.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash) VALUES($1,'test',$2,$3,'SUCCEEDED','input','hash')`, op, actorID, id); err != nil {
			t.Fatal(err)
		}
		snapshot := sealed
		if i == 2 {
			snapshot = []byte("intentionally corrupt ciphertext")
		}
		if _, err = db.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,warehouse_id,method,status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,provider_reference,created_at,sla_due_at) VALUES($1::uuid,$2,$1::text,'test',$3,$4,'COD','CONFIRMED_COD','COMMITTED',200,0,200,$5,'reservation',clock_timestamp()+interval '1 day',$1::text,$6,$7)`, id, op, actorID, warehouse, snapshot, time.Now().Add(-time.Duration(i)*time.Hour), time.Now().Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, `INSERT INTO order_items(order_id,sku,quantity,unit_price,line_total,product_name) VALUES($1,'SKU-1',2,100,200,'Frozen product')`, id); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, `UPDATE orders SET ready_authorized=true WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
	}
	page := func(r AdminOrderReader, cursor string, queue bool) map[string]any {
		v, e := s.AdminOrderPage(ctx, r, "", cursor, 1, queue)
		if e != nil {
			t.Fatal(e)
		}
		return v.(map[string]any)
	}
	r := reader("WH-A", "WH-B")
	for _, queue := range []bool{false, true} {
		first := page(r, "", queue)
		items := first["items"].([]adminOrderSummary)
		if len(items) != 1 || items[0].ID != ids[1] || items[0].ItemCount != 2 {
			t.Fatal("scope must filter before limit", first)
		}
		cursor := first["next_cursor"].(*string)
		if cursor == nil {
			t.Fatal("missing continuation")
		}
		last := page(r, *cursor, queue)
		if last["items"].([]adminOrderSummary)[0].ID != ids[2] || last["next_cursor"].(*string) != nil {
			t.Fatal("incorrect final page", last)
		}
		if _, e := s.AdminOrderPage(ctx, reader("WH-B", ""), "", *cursor, 1, queue); e == nil {
			t.Fatal("cursor crossed scope")
		}
		if _, e := s.AdminOrderPage(ctx, r, "", *cursor, 1, !queue); e == nil {
			t.Fatal("cursor crossed queue filter")
		}
	}
	if _, err = s.AdminOrderDetail(ctx, r, ids[0]); err == nil {
		t.Fatal("foreign order disclosed")
	}
	// The corrupt ciphertext proves masked reads never invoke decryption.
	masked, err := s.AdminOrderDetail(ctx, r, ids[2])
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(masked)
	if bytes.Contains(raw, []byte("Snapshot recipient")) || !bytes.Contains(raw, []byte(`"recipient":null`)) {
		t.Fatal("PII leaked", string(raw))
	}
	visible, err := s.AdminOrderDetail(ctx, reader("WH-A", "WH-A"), ids[1])
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(visible)
	if !bytes.Contains(raw, []byte("Snapshot recipient")) || !bytes.Contains(raw, []byte("Frozen product")) {
		t.Fatal("snapshot missing", string(raw))
	}
	if _, err = s.AdminOrderDetail(ctx, reader("WH-A", "WH-A"), ids[2]); err == nil {
		t.Fatal("authorized corrupt snapshot should fail")
	}
}
