package shippingprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	api "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/internal/persistence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var ErrIgnored = errors.New("irrelevant shipping notification")

type Adapter struct {
	Client api.ShippingServiceClient
	Token  string
}

func Dial(ctx context.Context, target, token, ca, serverName string, local bool) (*Adapter, *grpc.ClientConn, error) {
	if len(token) < 32 {
		return nil, nil, errors.New("Shipping token must contain at least 32 bytes")
	}
	var cred credentials.TransportCredentials
	if local {
		host, _, err := net.SplitHostPort(target)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) || ca != "" || serverName != "" {
			return nil, nil, errors.New("insecure Shipping allowed only on explicit loopback test endpoint")
		}
		cred = insecure.NewCredentials()
	} else {
		if ca == "" || serverName == "" {
			return nil, nil, errors.New("Shipping requires pinned CA and server name")
		}
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, nil, errors.New("invalid Shipping CA")
		}
		cred = credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})
	}
	deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(deadline, target, grpc.WithTransportCredentials(cred), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}
	return &Adapter{Client: api.NewShippingServiceClient(conn), Token: token}, conn, nil
}

func (a *Adapter) Revision(ctx context.Context, id string, version int64) (persistence.SourceEvidence, error) {
	if _, err := uuid.Parse(id); err != nil || version < 1 {
		return persistence.SourceEvidence{}, domain.E("INVALID_SHIPPING_REVISION", 422)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.Token)
	r, err := a.Client.GetShipmentRevision(ctx, &api.GetShipmentRevisionRequest{ShipmentId: id, ResourceVersion: version})
	if err != nil {
		return persistence.SourceEvidence{}, domain.E("SHIPPING_REVISION_UNAVAILABLE", 503)
	}
	if r.GetShipmentId() != id || r.GetResourceVersion() != version || r.GetReadyGeneration() < 1 || r.GetSealCode() == "" || r.GetOccurredAt() == nil || r.GetOccurredAt().CheckValid() != nil || r.GetOccurredAt().AsTime().After(time.Now().Add(time.Second)) {
		return persistence.SourceEvidence{}, domain.E("INVALID_SHIPPING_PROOF", 422)
	}
	return persistence.SourceEvidence{Order: r.GetOrderId(), Resource: id, Source: "shipping", Kind: r.GetKind(), Version: version, Generation: r.GetReadyGeneration(), Seal: r.GetSealCode(), Occurred: r.GetOccurredAt().AsTime(), Settled: r.GetSettlementVerified()}, nil
}

// Kafka is a notification, never authority for a delivery/settlement decision.
// Fetch the exact authenticated provider revision before creating an internal
// command; do not reconstruct historical evidence from current shipment state.
func (a *Adapter) Notification(ctx context.Context, raw []byte) (domain.Event, error) {
	var n struct {
		Spec    string    `json:"specversion"`
		ID      string    `json:"id"`
		Source  string    `json:"source"`
		Type    string    `json:"type"`
		Subject string    `json:"subject"`
		Time    time.Time `json:"time"`
		Data    struct {
			Shipment   string `json:"shipment_id"`
			Order      string `json:"order_id"`
			Version    int64  `json:"resource_version"`
			Generation int64  `json:"ready_generation"`
		} `json:"data"`
	}
	if len(raw) > 1<<20 || json.Unmarshal(raw, &n) != nil || n.Spec != "1.0" || n.Source != "https://omama.vn/services/shipping-service" {
		return domain.Event{}, domain.E("INVALID_SHIPPING_NOTIFICATION", 422)
	}
	kind := map[string]string{"vn.omama.shipping.shipment.created.v2": "SHIPMENT_CREATED", "vn.omama.shipping.shipment.handed-over.v2": "SHIPMENT_DISPATCHED", "vn.omama.shipping.shipment.delivered.v2": "SHIPMENT_DELIVERED", "vn.omama.shipping.shipment.delivery-failed.v2": "SHIPMENT_FAILED"}[n.Type]
	if kind == "" {
		return domain.Event{}, ErrIgnored
	}
	if _, err := uuid.Parse(n.ID); err != nil || n.ID == uuid.Nil.String() || n.Subject != "shipment_id:"+n.Data.Shipment || n.Time.IsZero() {
		return domain.Event{}, domain.E("INVALID_SHIPPING_NOTIFICATION", 422)
	}
	proof, err := a.Revision(ctx, n.Data.Shipment, n.Data.Version)
	if err != nil {
		return domain.Event{}, err
	}
	if proof.Order != n.Data.Order || proof.Kind != kind || proof.Generation != n.Data.Generation || !proof.Occurred.Equal(n.Time.UTC().Truncate(time.Microsecond)) {
		return domain.Event{}, domain.E("SHIPPING_NOTIFICATION_MISMATCH", 422)
	}
	return domain.Event{ID: n.ID, Source: "shipping", OrderID: proof.Order, ResourceID: proof.Resource, Type: kind, Version: proof.Version, Generation: proof.Generation, Seal: proof.Seal}, nil
}

func TokenFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return strings.TrimSpace(string(b)), err
}
