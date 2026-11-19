package grpcapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	common "github.com/omamx/shipping-service/internal/gen/common/v1"
	api "github.com/omamx/shipping-service/internal/gen/shipping/v2"
	"github.com/omamx/shipping-service/internal/repository/postgres"
	"github.com/omamx/shipping-service/internal/transport/httpapi"
	"github.com/omamx/shipping-service/migrations"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// Real Shipping gRPC/HTTP and PostgreSQL; carrier FAKE, no live Order or Kafka broker.
// The existing Web/Identity connected suite covers the separate Order client/event consumer.
func TestPostgresOrderWireAndCareReturnIntegration(t *testing.T) {
	dsn := os.Getenv("SHIPPING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SHIPPING_TEST_DATABASE_URL is required; CI provides isolated PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "shipping_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Execute the exact additive IAM seed in PostgreSQL as well as the portable SQL guard.
	if _, err = pool.Exec(ctx, `CREATE TABLE iam_permissions(permission_code text PRIMARY KEY,resource_code text,action_code text,risk_level text,scope_required boolean,description text);
		CREATE TABLE iam_roles(role_id text PRIMARY KEY,role_code text,is_enabled boolean,archived_at timestamptz);
		CREATE TABLE iam_role_permissions(role_id text REFERENCES iam_roles,permission_code text REFERENCES iam_permissions,PRIMARY KEY(role_id,permission_code));
		INSERT INTO iam_roles VALUES ('active','ADMIN',true,NULL),('disabled','ADMIN',false,NULL),('archived','ADMIN',true,now()),('other','FINANCE',true,NULL)`); err != nil {
		t.Fatal(err)
	}
	seed, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "identity-service", "src", "main", "resources", "db", "migration", "V23__seed_shipping_permissions.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = pool.Exec(ctx, string(seed)); err != nil {
			t.Fatal(err)
		}
	}
	var grantCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam_role_permissions WHERE role_id='active' AND permission_code='SHIPMENT_VIEW'`).Scan(&grantCount); err != nil || grantCount != 1 {
		t.Fatalf("IAM grant seed: %d %v", grantCount, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam_role_permissions WHERE role_id<>'active'`).Scan(&grantCount); err != nil || grantCount != 0 {
		t.Fatalf("inactive IAM role granted: %d %v", grantCount, err)
	}
	if _, err = migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if applied, err := migrations.Apply(ctx, pool); err != nil || len(applied) != 0 {
		t.Fatalf("migration replay: %v %v", applied, err)
	}
	repo := postgres.New(pool)
	svc := application.NewService(repo, carrier.NewRegistry(carrier.NewFake()), repo)
	const token = "shipping-integration-service-token-32-characters"
	const secret = "shipping-integration-webhook-secret"
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer(grpc.UnaryInterceptor(AuthUnaryInterceptor(token)))
	api.RegisterShippingServiceServer(server, NewV2Server(svc))
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///shipping", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := api.NewShippingServiceClient(conn)
	if _, err = client.TrackShipment(ctx, &api.TrackShipmentRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("service auth bypass: %v", err)
	}
	auth := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	address := &common.Address{RecipientName: "Test customer", PhoneNumber: "0905000000", StreetAddress: "1 Test Street", Ward: "Test ward", Province: "Hue", CountryCode: "VN"}
	weight := &common.Weight{DecimalValue: "1000", Unit: common.WeightUnit_WEIGHT_UNIT_GRAM}
	dimensions := &api.ParcelDimensions{LengthCm: 20, WidthCm: 20, HeightCm: 10}
	warehouse := "WH-HCM-01"
	quoteRequest := &api.CalculateShippingFeeRequest{ContractVersion: "order-3.2", IdempotencyKey: "integration-quote-00001", DestinationAddress: address, TotalWeight: weight, ParcelDimensions: dimensions, OriginWarehouseId: &warehouse}
	quoteRequest.RequestHash, err = canonicalMessageHash(quoteRequest.ProtoReflect())
	if err != nil {
		t.Fatal(err)
	}
	quote, err := client.CalculateShippingFee(auth, quoteRequest)
	if err != nil {
		t.Fatal(err)
	}
	if quote.GetQuoteId() == "" || quote.GetQuoteVersion() < 1 {
		t.Fatalf("missing bound quote: %+v", quote)
	}
	quoteID := quote.GetQuoteId()
	request := &api.CreateShipmentRequest{IdempotencyKey: "integration-shipment-00001", OrderId: uuid.NewString(), PackageId: uuid.NewString(), SealCode: "integration-seal", ReadyGeneration: 1, ItemsDigest: strings.Repeat("a", 64), OrderSnapshotVersion: 1, DestinationAddress: address, ParcelWeight: weight, ParcelDimensions: dimensions, Carrier: quote.GetCarrier(), ServiceCode: quote.GetServiceCode(), CheckoutShippingFee: quote.GetShippingFee(), QuoteId: &quoteID}
	created, err := client.CreateShipment(auth, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := client.CreateShipment(auth, request)
	if err != nil || replay.GetShipmentId() != created.GetShipmentId() {
		t.Fatalf("create replay: %v", err)
	}
	request.SealCode = "different-seal"
	if _, err = client.CreateShipment(auth, request); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("changed request reused key: %v", err)
	}
	request.SealCode = "integration-seal"
	shipmentID := created.GetShipmentId()
	initial, err := client.TrackShipment(auth, &api.TrackShipmentRequest{ShipmentId: &shipmentID})
	if err != nil {
		t.Fatal(err)
	}
	proofRequest := &api.GetShipmentRevisionRequest{ShipmentId: shipmentID, ResourceVersion: initial.GetResourceVersion()}
	proof, err := client.GetShipmentRevision(auth, proofRequest)
	if err != nil || proof.GetKind() != "SHIPMENT_CREATED" || proof.GetOrderId() != request.OrderId {
		t.Fatalf("creation proof: %v %v", proof, err)
	}

	handler := httpapi.NewRouter(svc, token, secret, "", zerolog.Nop())
	for _, endpoint := range []string{"/api/v1/admin/shipping/shipments", "/api/v1/internal/return-shipments/" + uuid.NewString()} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, endpoint, nil))
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("missing auth on %s: %d", endpoint, res.Code)
		}
	}
	call := func(method, endpoint, key string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, endpoint, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Internal-Token", token)
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, r)
		return res
	}
	if res := call("POST", "/api/v1/internal/shipments/"+created.GetShipmentId()+"/handover", "integration-handover-001", nil); res.Code != 200 {
		t.Fatalf("handover: %d %s", res.Code, res.Body)
	}
	payload, _ := json.Marshal(map[string]any{"event_id": uuid.NewString(), "tracking_code": created.GetTrackingCode(), "status": "DELIVERED", "event_at": time.Now().UTC()})
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "/api/v1/webhooks/logistics/fake", bytes.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Shipping-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, r)
		if res.Code != 200 {
			t.Fatalf("webhook replay: %d %s", res.Code, res.Body)
		}
	}
	tracked, err := client.TrackShipment(auth, &api.TrackShipmentRequest{ShipmentId: &shipmentID})
	if err != nil || !strings.HasSuffix(tracked.GetCurrentStatus().String(), "DELIVERED") {
		t.Fatalf("tracking: %v %v", tracked, err)
	}
	immutable, err := client.GetShipmentRevision(auth, proofRequest)
	if err != nil || immutable.GetKind() != proof.GetKind() {
		t.Fatalf("creation revision changed: %v", err)
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE topic='shipping.events.v2'").Scan(&count); err != nil || count != 3 {
		t.Fatalf("created/dispatched/delivered outbox: %d %v", count, err)
	}

	returnCommand := domain.CreateReturnShipmentCommand{ReturnCaseID: uuid.NewString(), OrderID: request.OrderId, PackageID: request.PackageId, CarrierCode: "FAKE", ServiceCode: "STANDARD", PickupAddress: domain.Recipient{Name: "Customer", Phone: "0905000000", Street: "1 Customer", Ward: "Ward", Province: "Hue", CountryCode: "VN"}, Destination: domain.Recipient{Name: "Warehouse", Phone: "0905000001", Street: "1 Warehouse", Ward: "Ward", Province: "Hue", CountryCode: "VN"}, Parcel: domain.Parcel{WeightGrams: 1000, LengthCM: 20, WidthCM: 20, HeightCM: 10}, ReasonCode: "DAMAGED"}
	body, _ := json.Marshal(returnCommand)
	for i := 0; i < 2; i++ {
		res := call("POST", "/api/v1/internal/return-shipments", "integration-care-return-01", body)
		if res.Code != 201 {
			t.Fatalf("Care return replay: %d %s", res.Code, res.Body)
		}
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM return_shipments WHERE cod_amount_vnd=0 AND fee_payer='SHOP'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("return policy: %d %v", count, err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE topic='shipping.return.events.v1'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("return outbox replay: %d %v", count, err)
	}
}
