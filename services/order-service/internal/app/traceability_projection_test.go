package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/fulfillmentprovider"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	trace "github.com/omamx/order-service/internal/gen/traceability/v1"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/traceabilityprovider"
	"github.com/omamx/order-service/migrations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type traceReadPeers struct {
	packing.UnimplementedFulfillmentPackageProofCandidateServer
	trace.UnimplementedBatchLinkResolverServer
	proof           *packing.PackageRevision
	broken, offline bool
}

func (p *traceReadPeers) GetPackageRevision(ctx context.Context, input *packing.GetPackageRevisionRequest) (*packing.PackageRevision, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer "+strings.Repeat("f", 40) || input.PackageId != p.proof.PackageId || input.SealedTaskVersion != p.proof.SealedTaskVersion {
		return nil, status.Error(codes.PermissionDenied, "wrong sealed locator")
	}
	result := proto.Clone(p.proof).(*packing.PackageRevision)
	if p.broken {
		result.PackedLines = nil
	}
	return result, nil
}
func (p *traceReadPeers) ResolveBatchLinks(ctx context.Context, input *trace.ResolveBatchLinksRequest) (*trace.ResolveBatchLinksResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer "+strings.Repeat("t", 40) {
		return nil, status.Error(codes.Unauthenticated, "wrong resolver caller")
	}
	if p.offline {
		return nil, status.Error(codes.Unavailable, "fixture offline")
	}
	url := "https://shop.example.test/trace/abcdefghijklmnopqrstuv"
	out := &trace.ResolveBatchLinksResponse{Total: int32(len(input.Items))}
	for i, binding := range input.Items {
		item := &trace.ResolvedBatchLink{Binding: binding, Status: trace.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE}
		if i == 0 {
			item.Status = trace.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE
			item.QrUrl = &url
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// This exercises the complete Order read against real persisted Core snapshots
// and gRPC wire peers. It does not simulate Checkout/Shipping acceptance.
func TestCustomerTraceabilityFrozenProjectionPostgresGRPC(t *testing.T) {
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
	schema := "trace_projection_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	config, _ := pgxpool.ParseConfig(dsn)
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	cipher, _ := persistence.NewCipher("trace-test", map[string][]byte{"trace-test": bytes.Repeat([]byte{31}, 32)})
	store, err := persistence.NewStore(db, cipher, bytes.Repeat([]byte{32}, 32))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "order-snapshot.v1.examples.json"))
	if err != nil {
		t.Fatal(err)
	}
	var examples struct {
		Cases []struct {
			Snapshot core.SnapshotInput `json:"snapshot"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}
	d := examples.Cases[1].Snapshot
	principal, customer := uuid.NewString(), uuid.NewString()
	d.Owner = core.Owner{Kind: "CUSTOMER", PrincipalID: principal, CustomerID: &customer}
	d.Items = d.Items[:1]
	d.Items[0].Quantity = 5
	d.Items[0].LineSubtotal = d.Items[0].UnitPrice * 5
	d.Money.Subtotal = d.Items[0].LineSubtotal
	d.Money.FinalAmount = d.Money.Subtotal + d.Money.ShippingFee
	d.Payment.CollectAmount = &d.Money.FinalAmount
	d.Shipping.PackageWeightGrams = d.Items[0].WeightGrams * 5
	selection := []core.SelectionLine{{SKU: d.Items[0].SKU, Quantity: 5}}
	d.Provenance.SelectionDigest, _ = core.HashSelection(selection)
	d.Provenance.ItemsDigest, _ = core.HashManifest(selection)
	quote, cart := uuid.NewString(), uuid.NewString()
	accepted, err := store.Accept(ctx, persistence.AcceptanceInput{Owner: d.Owner, Method: "COD", Key: uuid.NewString(), RequestHash: strings.Repeat("a", 64), QuoteID: quote, CartID: cart, CartRevision: 1, SelectionDigest: d.Provenance.SelectionDigest, QuoteHash: d.Provenance.QuoteHash, QuoteExpiresAt: time.Now().Add(time.Hour), AcceptedJSON: []byte(`{"fixture":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	var now time.Time
	if err = db.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	d.OrderID, d.OperationID, d.SnapshotID = accepted.PlannedOrderID, accepted.OperationID, uuid.NewString()
	d.Provenance.QuoteID, d.Provenance.CartID, d.Provenance.CartRevision = quote, cart, 1
	d.Provenance.AcceptedAt, d.Provenance.CanonicalizedAt, d.PlacedAt = accepted.AcceptedAt, now, now
	d.Catalog.EvaluatedAt, d.Shipping.ValidUntil = now, now.Add(time.Hour)
	if _, err = core.NewSnapshot(d); err != nil {
		t.Fatal("invalid test snapshot", err)
	}
	data, _ := json.Marshal(d)
	encrypted, _ := cipher.Seal("canonical:"+accepted.OperationID, data)
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	lease, job := uuid.NewString(), uuid.NewString()
	for _, statement := range []struct {
		SQL  string
		Args []any
	}{
		{`UPDATE checkout_operations SET status='PROCESSING',stage='PLACE',canonical_cipher=$2,canonical_hash=$3 WHERE id=$1`, []any{accepted.OperationID, encrypted, hash}},
		{`UPDATE workflows SET stage='PLACE' WHERE aggregate_id=$1`, []any{accepted.OperationID}},
		{`UPDATE jobs SET status='DONE' WHERE workflow_id IN(SELECT id FROM workflows WHERE aggregate_id=$1)`, []any{accepted.OperationID}},
		{`INSERT INTO jobs(id,kind,key,stage,status,cohort,workflow_id,input_version,input_cipher,input_hash,lane,lease_owner,lease_epoch,lease_until) SELECT $2,'ORDER_PLACE',$3,'PLACE','LEASED','order-core-1',id,1,$4,$5,'INTERACTIVE',$6,1,clock_timestamp()+interval '30 seconds' FROM workflows WHERE aggregate_id=$1`, []any{accepted.OperationID, job, "trace-place:" + accepted.OperationID, encrypted, hash, lease}},
	} {
		if _, err = db.Exec(ctx, statement.SQL, statement.Args...); err != nil {
			t.Fatal(err)
		}
	}
	reservation := core.ResourceProof{OrderID: d.OrderID, ReservationID: uuid.NewString(), ItemsDigest: d.Provenance.ItemsDigest, WarehouseID: d.WarehouseID, State: "COMMITTED", ResourceVersion: 1, DecisionID: uuid.NewString(), DecisionAt: now, ExpiresAt: now.Add(time.Hour)}
	if _, err = db.Exec(ctx, `UPDATE checkout_operations SET reservation_id=$2,reservation_expires_at=$3 WHERE id=$1`, accepted.OperationID, reservation.ReservationID, reservation.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Place(ctx, persistence.Placement{OperationID: accepted.OperationID, Fence: persistence.LeaseFence{JobID: job, Owner: lease, Epoch: 1}, Proof: reservation}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE orders SET ready_authorized=true,ready_generation=1 WHERE id=$1`, d.OrderID); err != nil {
		t.Fatal(err)
	}
	task, packageID := uuid.NewString(), uuid.NewString()
	seal := "TRACE-TEST-SEAL"
	for i, kind := range []string{"PACKING_ACCEPTED", "PACKING_SEALED"} {
		if err = db.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
			t.Fatal(err)
		}
		event := domain.Event{ID: uuid.NewString(), OrderID: d.OrderID, ResourceID: task, Source: "fulfillment", Type: kind, Version: int64(i + 1), Generation: 1}
		if i == 1 {
			event.Seal = seal
		}
		state, e := store.ApplySource(ctx, event, persistence.SourceEvidence{Order: event.OrderID, Resource: task, Source: event.Source, Kind: kind, Version: event.Version, Generation: 1, Seal: event.Seal, Occurred: now})
		if e != nil || state != "APPLIED" {
			t.Fatal("test sealed lifecycle", state, e)
		}
	}
	proof := &packing.PackageRevision{ContractVersion: "order-3.2", OrderId: d.OrderID, TaskId: task, PackageId: packageID, SealedTaskVersion: 2, ReadyGeneration: 1, ItemsDigest: string(d.Provenance.ItemsDigest), WarehouseId: d.WarehouseID, SealCode: seal, PackedAt: timestamppb.New(now), ImmutableVideoObjectKey: "fixture/video", VideoChecksumSha256: strings.Repeat("c", 64), Parcel: &packing.ParcelMeasurements{WeightGrams: 1000, LengthCm: 20, WidthCm: 20, HeightCm: 10}, PackedLines: []*packing.PackedBatchLine{{OrderLineId: d.Items[0].LineID, SkuCode: d.Items[0].SKU, BatchId: uuid.NewString(), BatchCode: "LOT-A", PackedQuantity: 3}, {OrderLineId: d.Items[0].LineID, SkuCode: d.Items[0].SKU, BatchId: uuid.NewString(), BatchCode: "LOT-B", PackedQuantity: 2}}}
	if _, err = db.Exec(ctx, `INSERT INTO fulfillment_shipping_handoffs(order_id,package_id,task_id,sealed_version,request_hash,request_cipher) VALUES($1,$2,$3,2,'fixture','fixture')`, d.OrderID, packageID, task); err != nil {
		t.Fatal(err)
	}
	peers := &traceReadPeers{proof: proof}
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	packing.RegisterFulfillmentPackageProofCandidateServer(server, peers)
	trace.RegisterBatchLinkResolverServer(server, peers)
	go server.Serve(listener)
	defer server.Stop()
	connection, err := grpc.NewClient("passthrough:///trace-projection", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	service := &Service{DB: db, CoreCheckout: &persistence.CheckoutRuntime{Store: store}, Payment: &persistence.Payment{Store: store}, Fulfillment: &fulfillmentprovider.Adapter{Packages: packing.NewFulfillmentPackageProofCandidateClient(connection), Token: strings.Repeat("f", 40)}, Traceability: &traceabilityprovider.Adapter{Client: trace.NewBatchLinkResolverClient(connection), Token: strings.Repeat("t", 40), PublicOrigin: "https://shop.example.test"}}
	actor := domain.Principal{Role: "CUSTOMER", ID: principal}
	result, err := service.CustomerTraceability(ctx, d.OrderID, actor)
	if err != nil || result.State != "READY" || result.Total != 2 {
		t.Fatal(result, err)
	}
	if result.Items[0].Quantity+result.Items[1].Quantity != 5 || result.Items[0].SKU != d.Items[0].SKU {
		t.Fatal("frozen quantities changed", result)
	}
	peers.offline = true
	result, err = service.CustomerTraceability(ctx, d.OrderID, actor)
	if err != nil || result.State != "READY" || result.Items[0].Status != "UNAVAILABLE" || result.Items[1].Status != "UNAVAILABLE" {
		t.Fatal(result, err)
	}
	peers.broken = true
	result, err = service.CustomerTraceability(ctx, d.OrderID, actor)
	if err != nil || result.State != "UNAVAILABLE" || result.Total != 0 {
		t.Fatal(result, err)
	}
	order, err := service.GetOrder(ctx, d.OrderID, actor, false)
	if err != nil || order.Status != "PACKED" || order.Version != 3 {
		t.Fatal("read changed lifecycle", order.Status, order.Version, err)
	}
	// Emulate an authenticated pre-PR176 frozen row without rewriting its
	// commercial fields or lifecycle. The reader must retain its original hash.
	root, err := store.LoadOrder(ctx, d.OrderID, actor.Scope())
	if err != nil {
		t.Fatal(err)
	}
	var historical map[string]any
	if err = json.Unmarshal(root.Snapshot().Bytes(), &historical); err != nil {
		t.Fatal(err)
	}
	historical["catalog"] = map[string]any{"snapshot_id": "historical-catalog-proof", "version": d.Catalog.CatalogVersion, "valid_until": d.Shipping.ValidUntil}
	historicalJSON, _ := json.Marshal(historical)
	frozen, err := core.DecodePersistedSnapshot(historicalJSON)
	if err != nil {
		t.Fatal(err)
	}
	frozenCipher, _ := cipher.Seal("snapshot:"+d.OrderID, frozen.Bytes())
	// Seed a historical encoding only in this disposable schema. Restore both
	// update guards in the same transaction before exercising production reads.
	seed, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Rollback(ctx)
	if _, err = seed.Exec(ctx, `ALTER TABLE orders DISABLE TRIGGER immutable_snapshot; ALTER TABLE orders DISABLE TRIGGER core_snapshot_identity`); err != nil {
		t.Fatal(err)
	}
	if _, err = seed.Exec(ctx, `UPDATE orders SET snapshot_cipher=$2,snapshot_hash=$3 WHERE id=$1`, d.OrderID, frozenCipher, string(frozen.Hash())); err != nil {
		t.Fatal(err)
	}
	if _, err = seed.Exec(ctx, `ALTER TABLE orders ENABLE TRIGGER immutable_snapshot; ALTER TABLE orders ENABLE TRIGGER core_snapshot_identity`); err != nil {
		t.Fatal(err)
	}
	if err = seed.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	peers.offline, peers.broken = false, false
	result, err = service.CustomerTraceability(ctx, d.OrderID, actor)
	if err != nil || result.State != "READY" || result.Total != 2 {
		t.Fatal("historical frozen source lookup failed", result, err)
	}
	root, err = store.LoadOrder(ctx, d.OrderID, actor.Scope())
	if err != nil || root.Snapshot().Hash() != frozen.Hash() || root.Version() != 3 {
		t.Fatal("historical lookup rewrote snapshot or lifecycle", err)
	}
	if _, err = db.Exec(ctx, `UPDATE orders SET snapshot_hash=$2 WHERE id=$1`, d.OrderID, strings.Repeat("0", 64)); err == nil {
		t.Fatal("historical fixture left immutable guards disabled")
	}
}
