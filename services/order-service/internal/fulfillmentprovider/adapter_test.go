package fulfillmentprovider

import (
	"context"
	"encoding/json"
	f "github.com/omamx/order-service/internal/gen/fulfillment/v1"
	p "github.com/omamx/order-service/internal/gen/packing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"strings"
	"testing"
	"time"
)

type proofFixture struct {
	f.UnimplementedFulfillmentServiceServer
	p.UnimplementedFulfillmentPackageProofCandidateServer
	task *f.TaskRevision
	pack *p.PackageRevision
}

func (s *proofFixture) GetTaskRevision(ctx context.Context, in *f.GetTaskRevisionRequest) (*f.TaskRevision, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("authorization")) != 1 || md.Get("authorization")[0] != "Bearer "+strings.Repeat("s", 40) {
		return nil, status.Error(codes.Unauthenticated, "wrong caller")
	}
	return s.task, nil
}
func (s *proofFixture) GetPackageRevision(ctx context.Context, in *p.GetPackageRevisionRequest) (*p.PackageRevision, error) {
	return s.pack, nil
}
func TestNotificationRequiresExactProviderProof(t *testing.T) {
	order := "5c45692d-8239-4afb-8944-bb833e69dbcd"
	task := "449d0983-e11c-49f9-a87b-765ea41705dd"
	pack := "9bcf198e-2102-46ce-be2b-79ecc15cfa93"
	now := time.Now().UTC().Truncate(time.Microsecond)
	server := &proofFixture{task: &f.TaskRevision{TaskId: task, OrderId: order, ResourceVersion: 7, ReadyGeneration: 1, Kind: "PACKING_SEALED", SealCode: "SEAL-123", OccurredAt: timestamppb.New(now)}, pack: &p.PackageRevision{ContractVersion: "order-3.2", TaskId: task, OrderId: order, PackageId: pack, SealedTaskVersion: 7, ReadyGeneration: 1, SealCode: "SEAL-123", ImmutableVideoObjectKey: "verified/file/hash", VideoChecksumSha256: strings.Repeat("a", 64), PackedAt: timestamppb.New(now), Parcel: &p.ParcelMeasurements{WeightGrams: 1250, LengthCm: 30, WidthCm: 20, HeightCm: 10}}}
	listener := bufconn.Listen(1 << 20)
	rpc := grpc.NewServer()
	f.RegisterFulfillmentServiceServer(rpc, server)
	p.RegisterFulfillmentPackageProofCandidateServer(rpc, server)
	go rpc.Serve(listener)
	defer rpc.Stop()
	conn, e := grpc.NewClient("passthrough:///fixture", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	adapter := &Adapter{Client: f.NewFulfillmentServiceClient(conn), Packages: p.NewFulfillmentPackageProofCandidateClient(conn), Token: strings.Repeat("s", 40)}
	envelope := func() map[string]any {
		return map[string]any{"specversion": "1.0", "id": "ec6e3fc7-a73a-4d37-9d1c-84d7003458d8", "source": "https://omama.vn/services/fulfillment-service", "subject": "order_id:" + order, "type": "vn.omama.fulfillment.packing.completed.v2", "time": now, "data": map[string]any{"order_id": order, "task_id": task, "resource_version": 7, "ready_generation": 1, "package_id": pack, "seal_code": "SEAL-123", "s3_video_object_key": "verified/file/hash", "video_checksum_sha256": strings.Repeat("a", 64)}}
	}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"original", func(map[string]any) {}},
		{"another-order", func(n map[string]any) {
			n["data"].(map[string]any)["order_id"] = "13ea2977-a460-4844-b4d5-af64d0678189"
		}},
		{"wrong-generation", func(n map[string]any) { n["data"].(map[string]any)["ready_generation"] = 2 }},
		{"changed-video", func(n map[string]any) { n["data"].(map[string]any)["video_checksum_sha256"] = strings.Repeat("b", 64) }},
		{"wrong-seal", func(n map[string]any) { n["data"].(map[string]any)["seal_code"] = "SEAL-OTHER" }},
		{"wrong-revision", func(n map[string]any) { n["data"].(map[string]any)["resource_version"] = 8 }},
		{"rewritten-time", func(n map[string]any) { n["time"] = now.Add(time.Second) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := envelope()
			tc.change(n)
			raw, _ := json.Marshal(n)
			out, e := adapter.Notification(context.Background(), raw)
			if tc.name == "original" {
				if e != nil || out.Package == nil || out.Package.Parcel.WeightGrams != 1250 {
					t.Fatalf("valid sealed proof failed: %v", e)
				}
			} else if e == nil {
				t.Fatal("unbound event accepted")
			}
		})
	}
	adapter.Token = strings.Repeat("x", 40)
	raw, _ := json.Marshal(envelope())
	if _, e = adapter.Notification(context.Background(), raw); e == nil {
		t.Fatal("unauthenticated proof accepted")
	}
}
