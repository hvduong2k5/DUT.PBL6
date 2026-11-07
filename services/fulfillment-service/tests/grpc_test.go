package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"github.com/omamx/fulfillment-service/internal/adapters"
	"github.com/omamx/fulfillment-service/internal/application"
	d "github.com/omamx/fulfillment-service/internal/domain"
	c "github.com/omamx/fulfillment-service/internal/gen/common/v1"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	o "github.com/omamx/fulfillment-service/internal/gen/order/v1"
	"github.com/omamx/fulfillment-service/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"testing"
	"time"
)

type fixtureAuthority struct{ in d.AdmissionInput }

func (a fixtureAuthority) Admission(context.Context, *f.AcceptTaskRequest) (d.Permit, d.Inputs, d.Stock, error) {
	return a.in.Permit, a.in.Inputs, a.in.Stock, nil
}
func (a fixtureAuthority) ValidatePacking(_ context.Context, t d.Task) (*d.Validation, error) {
	return validation(t), nil
}
func dial(t *testing.T, s *grpc.Server) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1048576)
	go s.Serve(listener)
	t.Cleanup(s.Stop)
	conn, e := grpc.NewClient("passthrough:///fixture", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn.Close(); listener.Close() })
	return conn
}
func TestGRPCAdmissionACLAndExactProof(t *testing.T) {
	store, in := fixture(t)
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	auth := transport.Auth{Key: &key.PublicKey, Issuer: "internal-fixture"}
	svc := &application.Service{Store: store, Authority: fixtureAuthority{in}, AdmissionEnabled: true}
	server := grpc.NewServer(grpc.UnaryInterceptor(auth.Interceptor))
	f.RegisterFulfillmentServiceServer(server, &transport.RPC{Service: svc, Auth: auth})
	client := f.NewFulfillmentServiceClient(dial(t, server))
	caller := func(service string) context.Context {
		claims := transport.Claims{Service: service, RegisteredClaims: jwt.RegisteredClaims{Subject: "service:" + service, Issuer: auth.Issuer, Audience: jwt.ClaimStrings{"fulfillment-internal"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
		token, e := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	}
	permit := in.Permit
	request := &f.AcceptTaskRequest{OrderId: in.Inputs.OrderID, Generation: 1, IdempotencyKey: in.Command.Key, ContractVersion: d.Contract, Permit: &c.FulfillmentPermit{PermitId: permit.ID, OrderId: permit.OrderID, Generation: permit.Generation, ItemsDigest: permit.Digest, WarehouseId: permit.Warehouse, OrderVersion: permit.OrderVersion, PolicyVersion: permit.Policy, ExpiresAt: timestamppb.New(permit.Expires), TaskRequestKey: permit.Key, Audience: permit.Audience}}
	request.RequestHash, _ = application.ProtoHash(request)
	if _, e = client.AcceptTask(caller("inventory-service"), request); status.Code(e) != codes.PermissionDenied {
		t.Fatal("wrong service accepted", e)
	}
	accepted, e := client.AcceptTask(caller("fulfillment-service"), request)
	if e != nil || accepted.State != "ACCEPTED" || accepted.Command == nil {
		t.Fatal("grpc accept", e)
	}
	again, e := client.AcceptTask(caller("fulfillment-service"), request)
	if e != nil || again.Command.DecisionId != accepted.Command.DecisionId {
		t.Fatal("grpc replay changed identity", e)
	}
	query, e := client.GetTaskDecision(caller("inventory-service"), &f.GetTaskDecisionRequest{OrderId: request.OrderId, Generation: 1, ContractVersion: d.Contract})
	if e != nil || query.State != "ACCEPTED" {
		t.Fatal("proof query", e)
	}
	rev, e := client.GetTaskRevision(caller("order-service"), &f.GetTaskRevisionRequest{TaskId: accepted.GetTaskId(), ResourceVersion: 1})
	if e != nil || rev.Kind != "PACKING_ACCEPTED" || rev.ItemsDigest != in.Inputs.Digest {
		t.Fatal("exact grpc proof", e)
	}
}
func TestMissingPeerCapabilityDoesNotFallback(t *testing.T) {
	server := grpc.NewServer()
	o.RegisterOrderServiceServer(server, &o.UnimplementedOrderServiceServer{})
	conn := dial(t, server)
	peers := adapters.NewPeers(conn, conn, "fixture-service-token")
	_, _, _, e := peers.Admission(context.Background(), &f.AcceptTaskRequest{Permit: &c.FulfillmentPermit{ExpiresAt: timestamppb.New(time.Now().Add(time.Minute))}})
	if status.Code(e) != codes.Unimplemented {
		t.Fatalf("missing real peer capability did not fail closed: %v", e)
	}
}
