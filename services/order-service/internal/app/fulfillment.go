package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/fulfillmentprovider"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	shipping "github.com/omamx/order-service/internal/gen/shipping/v2"
	"github.com/omamx/order-service/internal/shippingprovider"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"log/slog"
	"os"
	"strconv"
	"time"
)

func (s *Service) EnableFulfillmentFromEnv(ctx context.Context) error {
	target := os.Getenv("ORDER_FULFILLMENT_GRPC")
	if target == "" {
		return nil
	}
	if s.C.AutoComplete {
		return errors.New("real Fulfillment MVP requires ENABLE_AUTO_COMPLETE=false until Care/Finance completion authority is connected")
	}
	if s.C.Kafka == "" || s.Shipping == nil || s.CoreCheckout == nil {
		return errors.New("Fulfillment requires Kafka, core checkout and Shipping")
	}
	token, e := shippingprovider.TokenFile(os.Getenv("ORDER_FULFILLMENT_TOKEN_FILE"))
	if e != nil {
		return e
	}
	caller, e := shippingprovider.TokenFile(os.Getenv("ORDER_FULFILLMENT_CALLER_TOKEN_FILE"))
	if e != nil || len(caller) < 32 {
		return errors.New("separate Fulfillment caller credential required")
	}
	raw, e := os.ReadFile(os.Getenv("ORDER_PACKING_POLICY_FILE"))
	if e != nil {
		return e
	}
	policy := &packing.PackingPolicy{}
	if e = protojson.Unmarshal(raw, policy); e != nil {
		return e
	}
	if policy.PolicyVersion == "" || policy.EvidenceMode != "VIDEO_REQUIRED" || len(policy.Checklist) == 0 || policy.MaximumWeightGrams < 1 || policy.MaximumDimensionCm < 1 || policy.MaximumVideoBytes < 1 || policy.MaximumVideoSeconds < 1 {
		return errors.New("explicit versioned packing/video policy required")
	}
	adapter, conn, e := fulfillmentprovider.Dial(target, token, os.Getenv("ORDER_FULFILLMENT_CA_FILE"), os.Getenv("ORDER_FULFILLMENT_SERVER_NAME"), s.C.Mode == "sandbox" && os.Getenv("ORDER_FULFILLMENT_ALLOW_LOCAL_INSECURE") == "true")
	if e != nil {
		return e
	}
	adapter.CallerToken = caller
	adapter.Policy = policy
	s.Fulfillment = adapter
	prior := s.PaymentClose
	s.PaymentClose = func() error {
		if prior != nil {
			_ = prior()
		}
		return conn.Close()
	}
	return nil
}
func (s *Service) ProcessFulfillmentNotification(ctx context.Context, raw []byte) (string, error) {
	if s.Fulfillment == nil || s.CoreCheckout == nil {
		return "", domain.E("FULFILLMENT_PROVIDER_UNAVAILABLE", 503)
	}
	n, e := s.Fulfillment.Notification(ctx, raw)
	if errors.Is(e, fulfillmentprovider.ErrIgnored) {
		return "IGNORED", nil
	}
	if e != nil {
		return "", e
	}
	n.Event.Signature = domain.EventSignature(n.Event, s.C.InternalToken)
	state, e := s.ProcessEvent(ctx, n.Event)
	if e != nil {
		return "", e
	}
	if n.Package != nil && state != "REJECTED" {
		if e = s.queueFulfillmentShipment(ctx, n.Package); e != nil {
			return "", e
		}
	}
	return state, nil
}
func (s *Service) consumeFulfillment(ctx context.Context, topic string) {
	defer s.stop.Done()
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: splitBrokers(s.C.Kafka), Topic: topic, GroupID: consumerGroup(s.C) + "-" + topic, StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0})
	defer reader.Close()
	for ctx.Err() == nil {
		message, e := reader.FetchMessage(ctx)
		if e != nil {
			continue
		}
		for ctx.Err() == nil {
			_, e = s.ProcessFulfillmentNotification(ctx, message.Value)
			if e != nil {
				name, code := s.ErrorCode(e)
				if code >= 500 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						continue
					}
				}
				slog.Warn("Fulfillment notification rejected", "topic", topic, "code", name)
			}
			if e = reader.CommitMessages(ctx, message); e == nil {
				break
			}
		}
	}
}

// Persist the exact encrypted request before any carrier-affecting RPC. A lost
// response/lease retries these same bytes and package-derived idempotency key.
func (s *Service) queueFulfillmentShipment(ctx context.Context, p *packing.PackageRevision) error {
	var scope string
	if e := s.DB.QueryRow(ctx, `SELECT scope FROM orders WHERE id=$1`, p.OrderId).Scan(&scope); e != nil {
		return e
	}
	aggregate, e := s.Payment.Store.LoadOrder(ctx, p.OrderId, scope)
	if e != nil {
		return e
	}
	snapshot := aggregate.Snapshot().Data()
	if string(snapshot.Provenance.ItemsDigest) != p.ItemsDigest || snapshot.WarehouseID != p.WarehouseId || snapshot.Payment.Method != "VIETQR" {
		return domain.E("PACKING_ORDER_MISMATCH", 422)
	}
	if f := aggregate.Record().Fulfillment; f.TaskID != "" && (f.TaskID != p.TaskId || f.Generation != p.ReadyGeneration) {
		return domain.E("PACKING_TASK_MISMATCH", 422)
	}
	a := snapshot.Address
	parcel := p.Parcel
	request := &shipping.CreateShipmentRequest{IdempotencyKey: "fulfillment:shipment:" + p.PackageId, OrderId: p.OrderId, PackageId: p.PackageId, ReadyGeneration: p.ReadyGeneration, ItemsDigest: p.ItemsDigest, OrderSnapshotVersion: 1, SealCode: p.SealCode, DestinationAddress: &common.Address{RecipientName: a.RecipientName, PhoneNumber: a.PhoneE164, StreetAddress: a.Street, Ward: a.WardCode, Province: a.ProvinceCode, CountryCode: "VN"}, ParcelWeight: &common.Weight{DecimalValue: strconv.FormatInt(int64(parcel.WeightGrams), 10), Unit: common.WeightUnit_WEIGHT_UNIT_GRAM}, ParcelDimensions: &shipping.ParcelDimensions{LengthCm: parcel.LengthCm, WidthCm: parcel.WidthCm, HeightCm: parcel.HeightCm}, IsFragile: parcel.Fragile, Carrier: snapshot.Shipping.CarrierCode, ServiceCode: snapshot.Shipping.ServiceCode, CodAmount: &common.Money{CurrencyCode: "VND"}, CheckoutShippingFee: &common.Money{CurrencyCode: "VND", Units: snapshot.Money.ShippingFee}}
	bytes, e := proto.Marshal(request)
	if e != nil {
		return e
	}
	cipher, e := s.Seal(bytes)
	if e != nil {
		return e
	}
	hash := domain.Hash(request)
	_, e = s.DB.Exec(ctx, `INSERT INTO fulfillment_shipping_handoffs(order_id,package_id,task_id,sealed_version,request_hash,request_cipher) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(order_id) DO NOTHING`, p.OrderId, p.PackageId, p.TaskId, p.SealedTaskVersion, hash, cipher)
	if e != nil {
		return e
	}
	var pinned string
	if e = s.DB.QueryRow(ctx, `SELECT request_hash FROM fulfillment_shipping_handoffs WHERE order_id=$1`, p.OrderId).Scan(&pinned); e != nil {
		return e
	}
	if pinned != hash {
		return domain.E("HANDOFF_IDEMPOTENCY_CONFLICT", 409)
	}
	return nil
}
func (s *Service) WorkFulfillmentShipment(ctx context.Context) error {
	var id, task string
	var version, epoch int64
	var cipher []byte
	e := s.DB.QueryRow(ctx, `WITH pick AS(SELECT h.order_id FROM fulfillment_shipping_handoffs h JOIN orders o ON o.id=h.order_id WHERE h.shipment_id IS NULL AND h.retry_at<=clock_timestamp() AND (h.lease_until IS NULL OR h.lease_until<clock_timestamp()) AND ((o.status='PACKED' AND NOT o.operational_hold AND NOT o.has_active_case) OR o.shipment_id IS NOT NULL) ORDER BY h.created_at FOR UPDATE OF h SKIP LOCKED LIMIT 1) UPDATE fulfillment_shipping_handoffs h SET lease_owner=$1,lease_until=clock_timestamp()+interval '30 seconds',lease_epoch=lease_epoch+1,attempts=attempts+1 FROM pick WHERE h.order_id=pick.order_id RETURNING h.order_id::text,h.task_id::text,h.sealed_version,h.request_cipher,h.lease_epoch`, s.workerID).Scan(&id, &task, &version, &cipher, &epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	var payload []byte
	var shipment string
	e = s.Open(cipher, &payload)
	request := &shipping.CreateShipmentRequest{}
	if e == nil {
		e = proto.Unmarshal(payload, request)
	}
	associated := false
	if e == nil {
		var currentTask, currentSeal string
		var currentGeneration int64
		var bound *string
		e = s.DB.QueryRow(ctx, `SELECT task_id::text,seal_code,ready_generation,shipment_id::text FROM orders WHERE id=$1 AND ((status='PACKED' AND NOT operational_hold AND NOT has_active_case) OR shipment_id IS NOT NULL)`, id).Scan(&currentTask, &currentSeal, &currentGeneration, &bound)
		if e == nil && (currentTask != task || currentSeal != request.SealCode || currentGeneration != request.ReadyGeneration) {
			e = domain.E("HANDOFF_ORDER_MISMATCH", 422)
		}
		// This association was already applied from authenticated immutable
		// Shipping evidence (Kafka may win the race). Finish bookkeeping even
		// after handover; do not create another shipment or guess its revision.
		if e == nil && bound != nil {
			shipment = *bound
			associated = true
		}
	}
	if e == nil && !associated {
		proof, err := s.Fulfillment.Revision(ctx, task, version)
		e = err
		if e == nil && (proof.Order != id || proof.Kind != "PACKING_SEALED" || proof.Seal != request.SealCode || proof.Generation != request.ReadyGeneration || proof.Occurred.After(time.Now())) {
			e = domain.E("HANDOFF_SOURCE_NOT_READY", 503)
		}
	}
	if e == nil && !associated {
		shipment, e = s.Shipping.Create(ctx, request)
	}
	// Reconcile the provider's immutable creation revision even if Kafka is delayed.
	if e == nil && !associated {
		proof, err := s.Shipping.CreationRevision(ctx, request, shipment)
		e = err
		if e == nil {
			event := domain.Event{ID: uuid.NewSHA1(uuid.MustParse(request.PackageId), []byte("shipping-created")).String(), Source: "shipping", OrderID: id, ResourceID: shipment, Type: "SHIPMENT_CREATED", Version: proof.Version, Generation: proof.Generation, Seal: proof.Seal}
			event.Signature = domain.EventSignature(event, s.C.InternalToken)
			state, err := s.ProcessEvent(ctx, event)
			e = err
			if e == nil && state != "APPLIED" {
				e = domain.E("SHIPPING_ASSOCIATION_PENDING", 503)
			}
		}
	}
	_, writeErr := s.DB.Exec(ctx, `UPDATE fulfillment_shipping_handoffs SET shipment_id=CASE WHEN $4 THEN $5::uuid ELSE NULL END,lease_until=NULL,retry_at=clock_timestamp()+interval '5 seconds' WHERE order_id=$1 AND lease_owner=$2 AND lease_epoch=$3`, id, s.workerID, epoch, e == nil, func() any {
		if shipment == "" {
			return nil
		}
		return shipment
	}())
	if e != nil {
		return fmt.Errorf("fulfillment handoff pending: %w", e)
	}
	return writeErr
}
