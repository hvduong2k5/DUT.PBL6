package paymentinventory

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Catalog/Shipping terms are a validated fixture in this provider integration test.
// The checkout worker, encrypted Order database and Inventory are real.
type acceptedPeerTerms struct{}

func (acceptedPeerTerms) Revalidate(context.Context, core.SnapshotInput) error { return nil }

func testTerminalCheckoutWorker(t *testing.T, a Adapter) {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("PEER_ORDER_DATABASE_URL")
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "peer_order_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	cipher, err := persistence.NewCipher("peer", map[string][]byte{"peer": bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	store, err := persistence.NewStore(db, cipher, bytes.Repeat([]byte{8}, 32))
	if err != nil {
		t.Fatal(err)
	}
	s := terminalSnapshot(t)
	connection := uuid.NewString()
	if _, err = db.Exec(ctx, `INSERT INTO payment_provider_connections(id,provider,merchant_identity,environment,secret_ref) VALUES($1,'SEPAY',$2,'TEST','peer-fixture')`, connection, s.MerchantID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO payment_receiver_profiles(id,connection_id,bank_account_uuid,config_version,config_cipher,accepting_new_payments) VALUES($1,$2,$3,$4,$5,true)`, *s.Payment.ReceiverProfileID, connection, uuid.NewString(), *s.Payment.ReceiverProfileVersion, []byte("unused-receiver-fixture")); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(persistence.AcceptedCheckout{Terms: s, QuoteExpires: s.Catalog.ValidUntil})
	if err != nil {
		t.Fatal(err)
	}
	in := persistence.AcceptanceInput{Owner: s.Owner, Method: s.Payment.Method, Key: uuid.NewString(), RequestHash: strings.Repeat("a", 64), QuoteID: s.Provenance.QuoteID, CartID: s.Provenance.CartID, CartRevision: s.Provenance.CartRevision, SelectionDigest: s.Provenance.SelectionDigest, QuoteHash: s.Provenance.QuoteHash, QuoteExpiresAt: s.Catalog.ValidUntil, AcceptedJSON: raw}
	accepted, err := store.Accept(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	adapter := a
	adapter.Client = &lostTerminalACK{InventoryServiceClient: a.Client, reserve: true, commit: true}
	runtime := persistence.CheckoutRuntime{Store: store, Source: acceptedPeerTerms{}, Inventory: adapter}
	scope := strings.ToLower(s.Owner.Kind) + ":" + s.Owner.PrincipalID
	var completed persistence.Acceptance
	for i := 0; i < 8; i++ {
		_, err = runtime.Work(ctx)
		if err != nil && status.Code(err) != codes.Unavailable {
			t.Fatal("checkout worker", err)
		}
		completed, err = store.ReadAcceptance(ctx, accepted.OperationID, scope)
		if err != nil {
			t.Fatal(err)
		}
		if completed.Status == "SUCCEEDED" {
			break
		}
		if _, err = db.Exec(ctx, "UPDATE jobs SET run_after=clock_timestamp() WHERE status='RETRY_WAIT'"); err != nil {
			t.Fatal(err)
		}
	}
	if completed.Status != "SUCCEEDED" || completed.OrderID == nil {
		t.Fatal("checkout did not place", completed.Status)
	}
	replay, err := store.Accept(ctx, in)
	if err != nil || replay.OperationID != accepted.OperationID || replay.OrderID == nil || *replay.OrderID != *completed.OrderID {
		t.Fatal("durable checkout replay", err)
	}
	root, err := store.LoadOrder(ctx, *completed.OrderID, scope)
	if err != nil {
		t.Fatal("persisted Order projection", err)
	}
	key := "peer-payment-finalize-" + root.ID()
	if _, err = adapter.Finalize(ctx, root, key); status.Code(err) != codes.Unavailable {
		t.Fatal("finalize ACK injection", err)
	}
	proof, err := adapter.Finalize(ctx, root, key)
	if err != nil || proof.State != "COMMITTED" {
		t.Fatal("persisted Order finalize recovery", err)
	}
	if proof.DecisionAt.After(time.Now().Add(time.Minute)) {
		t.Fatal("invalid provider decision time")
	}
	t.Log("PASS durable acceptance -> worker reserve/retry -> persisted Order -> Inventory commit/recovery; Catalog/Shipping are fixture terms, payment receipt is outside this test")
}
