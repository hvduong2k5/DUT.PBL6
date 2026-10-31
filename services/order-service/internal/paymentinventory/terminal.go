package paymentinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/omamx/order-service/internal/domain/core"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// RequestHash matches the provider's order-3.2 profile: populated proto fields,
// snake_case names, integer JSON numbers, optional presence and sorted map keys.
// QueryAction must be blank when hashing a mutation. Transport/correlation/key
// identify the command registry separately and do not change the payload hash.
func RequestHash(m protoreflect.Message) string {
	var canonical func(protoreflect.Message) map[string]any
	canonical = func(m protoreflect.Message) map[string]any {
		out := map[string]any{}
		m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			name := string(f.Name())
			if name == "request_hash" || name == "idempotency_key" || name == "correlation_id" {
				return true
			}
			convert := func(v protoreflect.Value) any { return v.Interface() }
			if f.Kind() == protoreflect.MessageKind {
				convert = func(v protoreflect.Value) any { return canonical(v.Message()) }
			}
			if f.IsList() {
				a := []any{}
				for i := 0; i < v.List().Len(); i++ {
					a = append(a, convert(v.List().Get(i)))
				}
				out[name] = a
			} else {
				out[name] = convert(v)
			}
			return true
		})
		return out
	}
	raw, _ := json.Marshal(canonical(m))
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func (a Adapter) terminalContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if a.Client == nil || len(a.Token) < 32 {
		return nil, nil, ErrProof
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("x-internal-token")
	md.Set("authorization", "Bearer "+a.Token)
	return metadata.NewOutgoingContext(ctx, md), cancel, nil
}

func terminalRequest(s core.SnapshotInput, key string) (*inventory.ReserveStockRequest, error) {
	qty := map[string]int64{}
	for _, i := range s.Items {
		if i.Quantity <= 0 || i.Quantity > math.MaxInt32 || i.SKU == "" {
			return nil, ErrProof
		}
		for _, b := range []byte(i.SKU) {
			if b < 33 || b > 126 {
				return nil, ErrProof
			}
		}
		qty[i.SKU] += i.Quantity
		if qty[i.SKU] > math.MaxInt32 {
			return nil, ErrProof
		}
	}
	skus := []string{}
	for sku := range qty {
		skus = append(skus, sku)
	}
	sort.Strings(skus)
	if len(skus) == 0 {
		return nil, ErrProof
	}
	items := []*inventory.ReserveItem{}
	lines := []core.SelectionLine{}
	for _, sku := range skus {
		items = append(items, &inventory.ReserveItem{SkuCode: sku, Quantity: int32(qty[sku])})
		lines = append(lines, core.SelectionLine{SKU: sku, Quantity: qty[sku]})
	}
	digest, err := core.HashManifest(lines)
	if err != nil || digest != s.Provenance.ItemsDigest {
		return nil, ErrProof
	}
	r := &inventory.ReserveStockRequest{OrderId: s.OrderID, Items: items, TtlMinutes: 15, Channel: "D2C_WEB", PreferredWarehouseId: &s.WarehouseID, ContractVersion: "order-3.2", ItemsDigest: string(digest), IdempotencyKey: key, CorrelationId: s.OrderID}
	r.RequestHash = RequestHash(r.ProtoReflect())
	return r, nil
}

func (a Adapter) terminalReserve(ctx context.Context, s core.SnapshotInput, key string, create bool) (core.ResourceProof, error) {
	r, err := terminalRequest(s, key)
	if err != nil {
		return core.ResourceProof{}, err
	}
	ctx, cancel, err := a.terminalContext(ctx)
	if err != nil {
		return core.ResourceProof{}, err
	}
	defer cancel()
	out, err := a.Client.GetReservationByOrder(ctx, &inventory.ReservationCommand{OrderId: s.OrderID, ContractVersion: "order-3.2", IdempotencyKey: key, QueryAction: "ReserveStock"})
	if status.Code(err) == codes.NotFound && create {
		var response *inventory.ReserveStockResponse
		response, err = a.Client.ReserveStock(ctx, r)
		if err == nil {
			out = response.GetOutcome()
		}
	}
	if err != nil {
		return core.ResourceProof{}, err
	}
	return terminalProof(out, s, core.ResourceProof{}, key, r.RequestHash, "ReserveStock")
}

func (a Adapter) terminalCommand(ctx context.Context, s core.SnapshotInput, p core.ResourceProof, key, action string) (core.ResourceProof, error) {
	if p.ReservationID == "" || p.ResourceVersion < 1 {
		return core.ResourceProof{}, ErrProof
	}
	r := &inventory.ReservationCommand{OrderId: s.OrderID, ContractVersion: "order-3.2", IdempotencyKey: key, ExpectedReservationId: &p.ReservationID, ItemsDigest: string(s.Provenance.ItemsDigest), WarehouseId: s.WarehouseID, CorrelationId: s.OrderID}
	// Bind immutable reservation/digest/warehouse, not the latest observed version.
	// Omitting optional expected_version keeps this command identical after a lost
	// ACK and refreshed Lookup; provider terminal locking prevents resurrection.
	r.RequestHash = RequestHash(r.ProtoReflect())
	ctx, cancel, err := a.terminalContext(ctx)
	if err != nil {
		return core.ResourceProof{}, err
	}
	defer cancel()
	out, err := a.Client.GetReservationByOrder(ctx, &inventory.ReservationCommand{OrderId: s.OrderID, ContractVersion: "order-3.2", IdempotencyKey: key, QueryAction: action})
	if err == nil {
		return terminalProof(out, s, p, key, r.RequestHash, action)
	}
	if status.Code(err) != codes.NotFound {
		return core.ResourceProof{}, err
	}
	// NOT_FOUND here means no decision for this key, never proof of no resource.
	// Provider serializes the mutation with reserve/release/expiry under its lock.
	switch action {
	case "FinalizeReservation":
		out, err = a.Client.FinalizeReservation(ctx, r)
	case "ReleaseReservationByOrder":
		out, err = a.Client.ReleaseReservationByOrder(ctx, r)
	default:
		return core.ResourceProof{}, ErrProof
	}
	if err != nil {
		return core.ResourceProof{}, err
	}
	return terminalProof(out, s, p, key, r.RequestHash, action)
}

func terminalProof(out *inventory.ReservationOutcome, s core.SnapshotInput, p core.ResourceProof, key, hash, action string) (core.ResourceProof, error) {
	// Reuse resource binding checks but keep action names explicit per profile.
	logical := action
	if action == "ReserveStock" {
		logical = "RESERVE"
	}
	if out == nil || out.Command == nil || out.Command.Action != action || out.ObservedAt == nil || out.ObservedAt.CheckValid() != nil || out.ObservedAt.AsTime().After(time.Now().Add(time.Minute)) {
		return core.ResourceProof{}, ErrProof
	}
	// checked does not mutate the provider evidence; clone a projection
	// solely to use its existing resource/version/hash validation.
	projected := proto.Clone(out).(*inventory.ReservationOutcome)
	projected.Command.Action = logical
	return checked(projected, s, p, key, hash, logical)
}
