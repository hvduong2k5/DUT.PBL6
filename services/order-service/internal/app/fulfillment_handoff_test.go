package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
	"testing"
)

func TestHandoffFinishesAfterKafkaAssociationAndHandover(t *testing.T) {
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("explicit isolated PostgreSQL required")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := "handoff_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = migrations.Apply(ctx, db); e != nil {
		t.Fatal(e)
	}
	block, e := aes.NewCipher(bytes.Repeat([]byte{17}, 32))
	if e != nil {
		t.Fatal(e)
	}
	aead, e := cipher.NewGCM(block)
	if e != nil {
		t.Fatal(e)
	}
	s := &Service{DB: db, AEAD: aead, workerID: uuid.NewString()}
	id, op, actor, task, shipment, pkg := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	// Seed only transport bookkeeping projections: the association represents
	// an already-applied authenticated source event, not a new domain decision.
	if _, e = db.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash) VALUES($1,'test',$2,$3,'SUCCEEDED','input','hash')`, op, actor, id); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,method,status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,provider_reference,task_id,shipment_id,seal_code,ready_generation) VALUES($1::uuid,$2,$1::text,'test',$3,'VIETQR','PACKED','COMMITTED',1,0,1,'snapshot','reservation',clock_timestamp()+interval '1 day',$1::text,$4,$5,'SEAL-TEST',1)`, id, op, actor, task, shipment); e != nil {
		t.Fatal(e)
	}
	request := &shipping.CreateShipmentRequest{OrderId: id, PackageId: pkg, SealCode: "SEAL-TEST", ReadyGeneration: 1, IdempotencyKey: "fulfillment:shipment:" + pkg}
	raw, e := proto.Marshal(request)
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := s.Seal(raw)
	if e != nil {
		t.Fatal(e)
	}
	hash := domain.Hash(request)
	if _, e = db.Exec(ctx, `INSERT INTO fulfillment_shipping_handoffs(order_id,package_id,task_id,sealed_version,request_hash,request_cipher) VALUES($1,$2,$3,5,$4,$5)`, id, pkg, task, hash, sealed); e != nil {
		t.Fatal(e)
	}
	for _, state := range []string{"PACKED", "SHIPPED"} {
		if _, e = db.Exec(ctx, "UPDATE orders SET status=$2 WHERE id=$1", id, state); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(ctx, "UPDATE fulfillment_shipping_handoffs SET shipment_id=NULL,retry_at=clock_timestamp() WHERE order_id=$1", id); e != nil {
			t.Fatal(e)
		}
		// No remote peers are configured: Kafka has already supplied the proof.
		if e = s.WorkFulfillmentShipment(ctx); e != nil {
			t.Fatal(state, e)
		}
		var bound, pinned string
		if e = db.QueryRow(ctx, "SELECT shipment_id::text,request_hash FROM fulfillment_shipping_handoffs WHERE order_id=$1", id).Scan(&bound, &pinned); e != nil || bound != shipment || pinned != hash {
			t.Fatal("lost association or changed original request", state, e)
		}
	}
}
