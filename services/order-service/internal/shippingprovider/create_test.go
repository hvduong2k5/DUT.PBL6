package shippingprovider

import (
	"context"
	api "github.com/omamx/order-service/internal/gen/shipping/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

type creationPeer struct {
	revisionPeer
	track     *api.TrackShipmentResponse
	requested int64
}

func (p *creationPeer) TrackShipment(context.Context, *api.TrackShipmentRequest, ...grpc.CallOption) (*api.TrackShipmentResponse, error) {
	return p.track, nil
}
func (p *creationPeer) GetShipmentRevision(_ context.Context, in *api.GetShipmentRevisionRequest, _ ...grpc.CallOption) (*api.ShipmentRevision, error) {
	p.requested = in.ResourceVersion
	return p.revision, nil
}
func TestCreationUsesProviderRevisionAndExactHistory(t *testing.T) {
	const id = "91edf9e6-a062-4703-8845-913d69b81bd1"
	request := &api.CreateShipmentRequest{OrderId: "order", PackageId: "package", SealCode: "SEAL-TEST", ReadyGeneration: 1}
	peer := &creationPeer{revisionPeer: revisionPeer{revision: &api.ShipmentRevision{ShipmentId: id, OrderId: "order", PackageId: "package", SealCode: "SEAL-TEST", ReadyGeneration: 1, ResourceVersion: 2, Kind: "SHIPMENT_CREATED", OccurredAt: timestamppb.New(time.Now().Add(-time.Second))}}, track: &api.TrackShipmentResponse{ShipmentId: id, OrderId: "order", PackageId: "package", SealCode: "SEAL-TEST", ReadyGeneration: 1, ResourceVersion: 2}}
	adapter := &Adapter{Client: peer, Token: "test-credential"}
	proof, e := adapter.CreationRevision(context.Background(), request, id)
	if e != nil || proof.Version != 2 || peer.requested != 2 {
		t.Fatal("assumed creation revision one", proof, e)
	}
	peer.revision.Kind = "SHIPMENT_DELIVERED"
	if _, e = adapter.CreationRevision(context.Background(), request, id); e == nil {
		t.Fatal("latest delivery used as creation proof")
	}
	peer.revision.Kind = "SHIPMENT_CREATED"
	peer.track.PackageId = "other-package"
	if _, e = adapter.CreationRevision(context.Background(), request, id); e == nil {
		t.Fatal("wrong package accepted")
	}
}
