package fulfillmentprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	fulfillment "github.com/omamx/order-service/internal/gen/fulfillment/v1"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	"github.com/omamx/order-service/internal/persistence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"net"
	"os"
	"time"
)

var ErrIgnored = errors.New("irrelevant fulfillment notification")

type Adapter struct {
	Client             fulfillment.FulfillmentServiceClient
	Packages           packing.FulfillmentPackageProofCandidateClient
	Token, CallerToken string
	Policy             *packing.PackingPolicy
}

func Dial(target, token, ca, serverName string, local bool) (*Adapter, *grpc.ClientConn, error) {
	if len(token) < 32 {
		return nil, nil, errors.New("Fulfillment credential must contain at least 32 bytes")
	}
	var transport credentials.TransportCredentials
	if local {
		host, _, e := net.SplitHostPort(target)
		ip := net.ParseIP(host)
		if e != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) || ca != "" || serverName != "" {
			return nil, nil, errors.New("insecure Fulfillment requires an explicit loopback TEST endpoint")
		}
		transport = insecure.NewCredentials()
	} else {
		raw, e := os.ReadFile(ca)
		if e != nil || serverName == "" {
			return nil, nil, errors.New("Fulfillment requires pinned CA and server name")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(raw) {
			return nil, nil, errors.New("invalid Fulfillment CA")
		}
		transport = credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})
	}
	// Nonblocking: peers may start independently; individual RPCs have deadlines.
	conn, e := grpc.NewClient(target, grpc.WithTransportCredentials(transport))
	if e != nil {
		return nil, nil, e
	}
	return &Adapter{Client: fulfillment.NewFulfillmentServiceClient(conn), Packages: packing.NewFulfillmentPackageProofCandidateClient(conn), Token: token}, conn, nil
}
func (a *Adapter) call(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.Token), cancel
}
func (a *Adapter) Revision(ctx context.Context, id string, version int64) (persistence.SourceEvidence, error) {
	if _, e := uuid.Parse(id); e != nil || version < 1 {
		return persistence.SourceEvidence{}, domain.E("INVALID_FULFILLMENT_REVISION", 422)
	}
	c, cancel := a.call(ctx)
	defer cancel()
	r, e := a.Client.GetTaskRevision(c, &fulfillment.GetTaskRevisionRequest{TaskId: id, ResourceVersion: version})
	if e != nil {
		return persistence.SourceEvidence{}, domain.E("FULFILLMENT_REVISION_UNAVAILABLE", 503)
	}
	if r.TaskId != id || r.ResourceVersion != version || r.ReadyGeneration < 1 || r.OccurredAt == nil || r.OccurredAt.CheckValid() != nil || (r.Kind != "PACKING_ACCEPTED" && r.Kind != "PACKING_SEALED") {
		return persistence.SourceEvidence{}, domain.E("INVALID_FULFILLMENT_PROOF", 422)
	}
	return persistence.SourceEvidence{Order: r.OrderId, Resource: r.TaskId, Source: "fulfillment", Kind: r.Kind, Version: r.ResourceVersion, Generation: r.ReadyGeneration, Seal: r.SealCode, Occurred: r.OccurredAt.AsTime()}, nil
}
func (a *Adapter) Package(ctx context.Context, task, id string, version int64) (*packing.PackageRevision, error) {
	if _, e := uuid.Parse(id); e != nil || version < 1 {
		return nil, domain.E("INVALID_PACKAGE_REVISION", 422)
	}
	c, cancel := a.call(ctx)
	defer cancel()
	r, e := a.Packages.GetPackageRevision(c, &packing.GetPackageRevisionRequest{PackageId: id, SealedTaskVersion: version})
	if e != nil {
		return nil, domain.E("PACKAGE_REVISION_UNAVAILABLE", 503)
	}
	hash, e := hex.DecodeString(r.GetVideoChecksumSha256())
	if e != nil || len(hash) != 32 || r.TaskId != task || r.PackageId != id || r.SealedTaskVersion != version || r.ReadyGeneration < 1 || r.ContractVersion != "order-3.2" || r.SealCode == "" || r.ImmutableVideoObjectKey == "" || r.Parcel == nil || r.Parcel.WeightGrams < 1 || r.Parcel.LengthCm < 1 || r.Parcel.WidthCm < 1 || r.Parcel.HeightCm < 1 || r.PackedAt == nil || r.PackedAt.CheckValid() != nil {
		return nil, domain.E("INVALID_PACKAGE_PROOF", 422)
	}
	return r, nil
}

type Notification struct {
	Event   domain.Event
	Package *packing.PackageRevision
}

func (a *Adapter) Notification(ctx context.Context, raw []byte) (Notification, error) {
	var n struct {
		Spec    string    `json:"specversion"`
		ID      string    `json:"id"`
		Source  string    `json:"source"`
		Type    string    `json:"type"`
		Subject string    `json:"subject"`
		Time    time.Time `json:"time"`
		Data    struct {
			Order      string `json:"order_id"`
			Task       string `json:"task_id"`
			Version    int64  `json:"resource_version"`
			Generation int64  `json:"ready_generation"`
			Package    string `json:"package_id"`
			Seal       string `json:"seal_code"`
			Video      string `json:"s3_video_object_key"`
			Checksum   string `json:"video_checksum_sha256"`
		} `json:"data"`
	}
	if len(raw) > 1<<20 || json.Unmarshal(raw, &n) != nil || n.Spec != "1.0" || n.Source != "https://omama.vn/services/fulfillment-service" {
		return Notification{}, domain.E("INVALID_FULFILLMENT_NOTIFICATION", 422)
	}
	kind := map[string]string{"vn.omama.fulfillment.task.accepted.v1": "PACKING_ACCEPTED", "vn.omama.fulfillment.packing.completed.v2": "PACKING_SEALED"}[n.Type]
	if kind == "" {
		return Notification{}, ErrIgnored
	}
	if id, e := uuid.Parse(n.ID); e != nil || id == uuid.Nil || n.Subject != "order_id:"+n.Data.Order || n.Time.IsZero() {
		return Notification{}, domain.E("INVALID_FULFILLMENT_NOTIFICATION", 422)
	}
	proof, e := a.Revision(ctx, n.Data.Task, n.Data.Version)
	if e != nil {
		return Notification{}, e
	}
	if proof.Order != n.Data.Order || proof.Generation != n.Data.Generation || proof.Kind != kind || !proof.Occurred.Equal(n.Time.UTC().Truncate(time.Microsecond)) {
		return Notification{}, domain.E("FULFILLMENT_NOTIFICATION_MISMATCH", 422)
	}
	out := Notification{Event: domain.Event{ID: n.ID, Source: "fulfillment", OrderID: proof.Order, ResourceID: proof.Resource, Type: proof.Kind, Version: proof.Version, Generation: proof.Generation, Seal: proof.Seal}}
	if kind == "PACKING_SEALED" {
		p, e := a.Package(ctx, proof.Resource, n.Data.Package, proof.Version)
		if e != nil {
			return Notification{}, e
		}
		if p.OrderId != proof.Order || p.ReadyGeneration != proof.Generation || p.SealCode != proof.Seal || n.Data.Seal != proof.Seal || p.ImmutableVideoObjectKey != n.Data.Video || p.VideoChecksumSha256 != n.Data.Checksum || !p.PackedAt.AsTime().Equal(proof.Occurred) {
			return Notification{}, domain.E("PACKAGE_NOTIFICATION_MISMATCH", 422)
		}
		out.Package = p
	}
	return out, nil
}
