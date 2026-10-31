package persistence

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/sepay"
	"time"
)

func (s *Store) paymentEvent(ctx context.Context, tx pgx.Tx, kind, id, topic, eventType, schemaRef, schemaHash string, version int64, ordinal int, data any, order any, at time.Time) error {
	stream, event := uuid.NewString(), uuid.NewString()
	var sequence int64
	err := tx.QueryRow(ctx, `INSERT INTO event_streams(id,aggregate_type,aggregate_id,topic,partition_key,routing_version,last_sequence) VALUES($1,$2,$3::uuid,$4,$3::text,1,1) ON CONFLICT(aggregate_type,aggregate_id) DO UPDATE SET last_sequence=event_streams.last_sequence+1 RETURNING id,last_sequence`, stream, kind, id, topic).Scan(&stream, &sequence)
	if err != nil {
		return err
	}
	subject := "order:" + id
	if kind == "RECEIPT" {
		subject = "receipt:" + id
	}
	if kind == "REFUND" {
		subject = "refund:" + id
	}
	payload, err := json.Marshal(map[string]any{"specversion": "1.0", "id": event, "source": "https://omama.vn/services/order-service", "type": eventType, "subject": subject, "time": at.UTC().Format(time.RFC3339Nano), "datacontenttype": "application/json", "data": data})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(event_id,stream_id,aggregate_type,aggregate_id,aggregate_version,ordinal,stream_sequence,order_id,event_type,schema_ref,schema_hash,payload_bytes,payload_hash,headers,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'{}',$14)`, event, stream, kind, id, version, ordinal, sequence, order, eventType, schemaRef, schemaHash, payload, hash(payload), at)
	return err
}
func (p *Payment) receiptEvent(ctx context.Context, tx pgx.Tx, id string, t sepay.Transaction, order any, evidenceHash string, at time.Time) error {
	bankAt, _ := sepay.BankTime(t.Date)
	data := map[string]any{"aggregate_id": id, "version": 1, "ordinal": 0, "receipt_id": id, "connection_id": p.Connection, "provider_transaction_id": t.ID, "order_id": order, "amount": t.In, "currency": "VND", "bank_occurred_at": bankAt, "evidence_hash": evidenceHash}
	return p.Store.paymentEvent(ctx, tx, "RECEIPT", id, "payment.events.v1", "vn.omama.payment.receipt_recorded.v1", "https://contracts.omama.example/payment/candidate/events.v1.schema.json", "bf4374af8940203097b703aa80f3c31a7b8dd3aa528518a9a993e04ffd7ff39a", 1, 0, data, order, at)
}
func (s *Store) paidEvent(ctx context.Context, tx pgx.Tx, a core.Aggregate, at time.Time) error {
	r, d := a.Record(), a.Snapshot().Data()
	data := map[string]any{"aggregate_id": a.ID(), "version": a.Version(), "ordinal": 0, "status": "PAID", "payment_status": "CONFIRMED", "stock_status": "COMMITTED", "items_digest": d.Provenance.ItemsDigest, "warehouse_id": d.WarehouseID, "reservation_id": r.Reservation.ReservationID, "stock_resource_version": r.Reservation.ResourceVersion, "allocation_id": r.Allocation.AllocationID, "stock_decision_id": r.Reservation.DecisionID, "voucher_finalization": nil}
	return s.paymentEvent(ctx, tx, "ORDER", a.ID(), "order.events.v3", "vn.omama.order.paid.v3", "https://contracts.omama.example/order/v3/paid.event.json", "e77a0a724ce30e27c00dc658ffb3818254db595f8de9d8b4c7bc27523f7cda1d", a.Version(), 0, data, a.ID(), at)
}
