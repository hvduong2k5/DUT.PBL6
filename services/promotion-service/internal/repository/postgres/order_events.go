package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

const orderPaidV3Type = "vn.omama.order.paid.v3"

var ErrInvalidOrderEvent = errors.New("invalid Order event")

var traceparentPattern = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type OrderPaidEvent struct {
	SpecVersion string        `json:"specversion"`
	ID          string        `json:"id"`
	Source      string        `json:"source"`
	Type        string        `json:"type"`
	Subject     string        `json:"subject"`
	Time        string        `json:"time"`
	DataContent string        `json:"datacontenttype"`
	Traceparent string        `json:"traceparent"`
	Data        OrderPaidData `json:"data"`
}

type OrderPaidData struct {
	AggregateID     string               `json:"aggregate_id"`
	Version         int64                `json:"version"`
	Ordinal         int64                `json:"ordinal"`
	Status          string               `json:"status"`
	Payment         string               `json:"payment_status"`
	Stock           string               `json:"stock_status"`
	ItemsDigest     string               `json:"items_digest"`
	WarehouseID     string               `json:"warehouse_id"`
	ReservationID   string               `json:"reservation_id"`
	StockVersion    int64                `json:"stock_resource_version"`
	AllocationID    string               `json:"allocation_id"`
	StockDecisionID string               `json:"stock_decision_id"`
	Voucher         *VoucherFinalization `json:"voucher_finalization"`
}

type VoucherFinalization struct {
	HoldID     string `json:"hold_id"`
	DecisionID string `json:"decision_id"`
}

// ProcessOrderPaidV3 atomically records the source event and queues voucher.used.
// A paid event never finalizes or consumes a voucher: that effect must already
// have been committed by FinalizeVoucherHold before Order emits this proof.
func (r *VoucherRepository) ProcessOrderPaidV3(ctx context.Context, raw []byte, topic string) (bool, error) {
	var event OrderPaidEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return false, fmt.Errorf("%w: decode paid event: %v", ErrInvalidOrderEvent, err)
	}
	if event.Type == "" {
		return false, fmt.Errorf("%w: event type is required", ErrInvalidOrderEvent)
	}
	if event.Type != orderPaidV3Type {
		return false, nil
	}
	var dataShape struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &dataShape); err != nil {
		return false, fmt.Errorf("%w: decode event data shape: %v", ErrInvalidOrderEvent, err)
	}
	if event.SpecVersion != "1.0" || event.ID == "" || event.Source != "https://omama.vn/services/order-service" ||
		event.Type != orderPaidV3Type || event.DataContent != "application/json" || event.Data.AggregateID == "" ||
		!uuidPattern.MatchString(event.ID) || !uuidPattern.MatchString(event.Data.AggregateID) ||
		event.Subject != "order:"+event.Data.AggregateID || event.Data.Version < 1 || event.Data.Ordinal < 0 ||
		event.Data.Status != "PAID" || event.Data.Payment != "CONFIRMED" || event.Data.Stock != "COMMITTED" ||
		!digestPattern.MatchString(event.Data.ItemsDigest) || event.Data.WarehouseID == "" ||
		!uuidPattern.MatchString(event.Data.ReservationID) || event.Data.StockVersion < 1 ||
		!uuidPattern.MatchString(event.Data.AllocationID) || event.Data.StockDecisionID == "" ||
		dataShape.Data["voucher_finalization"] == nil || (event.Traceparent != "" && !traceparentPattern.MatchString(event.Traceparent)) {
		return false, fmt.Errorf("%w: invalid paid v3 envelope or lifecycle proof", ErrInvalidOrderEvent)
	}
	if _, err := time.Parse(time.RFC3339Nano, event.Time); err != nil {
		return false, fmt.Errorf("%w: invalid event time: %v", ErrInvalidOrderEvent, err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	payloadHash := sha256.Sum256(raw)
	payloadHashText := hex.EncodeToString(payloadHash[:])
	tag, err := tx.Exec(ctx, `INSERT INTO promotion_consumed_events(event_id,event_type,aggregate_id,payload_hash)
		VALUES($1,$2,$3,$4) ON CONFLICT(event_id) DO NOTHING`, event.ID, event.Type, event.Data.AggregateID, payloadHashText)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		var storedType, storedAggregateID, storedHash string
		if err := tx.QueryRow(ctx, `SELECT event_type,aggregate_id,payload_hash FROM promotion_consumed_events WHERE event_id=$1`, event.ID).Scan(&storedType, &storedAggregateID, &storedHash); err != nil {
			return false, err
		}
		if storedType != event.Type || storedAggregateID != event.Data.AggregateID || storedHash != payloadHashText {
			return false, fmt.Errorf("%w: event ID was reused with different content", ErrInvalidOrderEvent)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if event.Data.Voucher == nil {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}
	proof := event.Data.Voucher
	if proof.HoldID == "" || proof.DecisionID == "" {
		return false, fmt.Errorf("%w: voucher finalization proof requires hold_id and decision_id", ErrInvalidOrderEvent)
	}
	var holdID, orderID, customerID, voucherCode, state, currency string
	var discount int64
	var usedAt time.Time
	err = tx.QueryRow(ctx, `SELECT h.id::text,h.order_id,h.customer_id,c.code,h.state,h.currency_code,h.discount_amount,u.finalized_at
		FROM voucher_holds h JOIN coupons c ON c.id=h.coupon_id JOIN voucher_usage u ON u.hold_id=h.id
		WHERE h.id=$1::uuid FOR UPDATE OF h`, proof.HoldID).
		Scan(&holdID, &orderID, &customerID, &voucherCode, &state, &currency, &discount, &usedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("%w: event references an unknown or unfinalized voucher hold", ErrInvalidOrderEvent)
	}
	if err != nil {
		return false, err
	}
	if orderID != event.Data.AggregateID || state != "FINALIZED" || proof.DecisionID != holdID {
		return false, fmt.Errorf("%w: voucher proof does not match the finalized hold", ErrInvalidOrderEvent)
	}
	traceparent := event.Traceparent
	if !traceparentPattern.MatchString(traceparent) {
		traceparent, err = newTraceparent()
		if err != nil {
			return false, err
		}
	}
	now := time.Now().UTC()
	payload, err := json.Marshal(map[string]any{
		"specversion":     "1.0",
		"id":              event.ID + ".voucher-used",
		"source":          "https://omama.vn/services/promotion-service",
		"type":            "vn.omama.promotion.voucher.used.v1",
		"subject":         "customer:" + customerID,
		"time":            now.Format(time.RFC3339Nano),
		"datacontenttype": "application/json",
		"traceparent":     traceparent,
		"data": map[string]any{
			"voucher_code":    voucherCode,
			"order_id":        orderID,
			"customer_id":     customerID,
			"discount_amount": map[string]any{"currency_code": currency, "units": discount, "nanos": 0},
			"used_at":         usedAt.UTC().Format(time.RFC3339Nano),
		},
	})
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(aggregate_type,aggregate_id,event_type,event_version,payload,occurred_at,topic,partition_key)
		VALUES('voucher_hold',$1,'vn.omama.promotion.voucher.used.v1',1,$2,$3,$4,$5)`, holdID, payload, now, topic, customerID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func newTraceparent() (string, error) {
	var traceID [16]byte
	var spanID [8]byte
	if _, err := rand.Read(traceID[:]); err != nil {
		return "", err
	}
	if _, err := rand.Read(spanID[:]); err != nil {
		return "", err
	}
	return "00-" + hex.EncodeToString(traceID[:]) + "-" + hex.EncodeToString(spanID[:]) + "-01", nil
}
