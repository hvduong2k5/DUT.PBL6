package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidOutboxEvent = errors.New("invalid outbox event parameters")
)

// OutboxStatus represents the processing state of an outbox record.
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusPublished OutboxStatus = "PUBLISHED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

// Canonical Event Types matching packages/events/schemas/
const (
	EventTypeStockReserved  = "vn.omama.inventory.stock.reserved.v1"
	EventTypeStockReleased  = "vn.omama.inventory.stock.released.v1"
	EventTypeStockCommitted = "vn.omama.inventory.stock.committed.v1"
	EventTypeStockDeducted  = "vn.omama.inventory.stock.committed.v1"
	EventTypeExpiryWarning  = "vn.omama.inventory.batch.expiry.warning.v1"
	EventTypeOrderPaid     = "vn.omama.order.paid.v1"

	TopicInventoryEvents = "inventory.events.v1"
	TopicOrderEvents     = "order.events.v1"

	EventSourceInventoryService = "https://omama.vn/services/inventory-service"
	CloudEventsSpecVersion      = "1.0"
	DefaultContentTypeJSON      = "application/json"
	DefaultTraceparent          = "00-00000000000000000000000000000000-0000000000000000-01"
)

// CloudEvent represents a standard CNCF CloudEvents 1.0 JSON envelope.
type CloudEvent[T any] struct {
	SpecVersion     string `json:"specversion"`
	ID              string `json:"id"`
	Source          string `json:"source"`
	Type            string `json:"type"`
	Subject         string `json:"subject"`
	Time            string `json:"time"`
	DataContentType string `json:"datacontenttype"`
	Traceparent     string `json:"traceparent"`
	Data            T      `json:"data"`
}

// StockReservedEventData defines the schema for vn.omama.inventory.stock.reserved.v1
type StockReservedEventData struct {
	ReservationID      string `json:"reservation_id"`
	OrderID            string `json:"order_id"`
	SKUCode            string `json:"sku_code"`
	Quantity           int    `json:"quantity"`
	WarehouseID        string `json:"warehouse_id"`
	AvailableQtyAfter  int    `json:"available_qty_after"`
	ExpiresAt          string `json:"expires_at"`
}

// StockReleasedEventData defines the schema for vn.omama.inventory.stock.released.v1
type StockReleasedEventData struct {
	ReservationID      string `json:"reservation_id"`
	OrderID            string `json:"order_id"`
	SKUCode            string `json:"sku_code"`
	QuantityReleased   int    `json:"quantity_released"`
	WarehouseID        string `json:"warehouse_id"`
	AvailableQtyAfter  int    `json:"available_qty_after"`
	ReleaseReason      string `json:"release_reason"`
}

// StockCommittedItem defines an item line in StockCommittedEventData.
type StockCommittedItem struct {
	SKUCode  string `json:"sku_code"`
	Quantity int    `json:"quantity"`
}

// StockCommittedEventData defines the schema for vn.omama.inventory.stock.committed.v1
type StockCommittedEventData struct {
	ReservationID string               `json:"reservation_id"`
	OrderID       string               `json:"order_id"`
	WarehouseID   string               `json:"warehouse_id,omitempty"`
	Items         []StockCommittedItem `json:"items"`
}

// ExpiryWarningEventData defines the schema for vn.omama.inventory.batch.expiry.warning.v1
type ExpiryWarningEventData struct {
	BatchCode         string `json:"batch_code"`
	SKUCode           string `json:"sku_code"`
	WarehouseID       string `json:"warehouse_id"`
	RemainingQuantity int    `json:"remaining_quantity"`
	BestBefore        string `json:"best_before"`
	DaysUntilExpiry   int    `json:"days_until_expiry"`
	RecommendedAction string `json:"recommended_action"`
	DetectedAt        string `json:"detected_at"`
}

// OrderPaidItem defines an item line in OrderPaidEventData.
type OrderPaidItem struct {
	SKUCode  string `json:"sku_code"`
	Quantity int    `json:"quantity"`
}

// OrderPaidEventData defines relevant data parsed from vn.omama.order.paid.v1
type OrderPaidEventData struct {
	OrderID   string          `json:"order_id"`
	Items     []OrderPaidItem `json:"items,omitempty"`
	Channel   string          `json:"channel,omitempty"`
}

// OutboxEvent maps to the outbox_events PostgreSQL table.
type OutboxEvent struct {
	ID            uuid.UUID       `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Status        OutboxStatus    `json:"status"`
	RetryCount    int             `json:"retry_count"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
	ErrorMessage  *string         `json:"error_message,omitempty"`
}

// NewOutboxEvent constructs a new pending OutboxEvent.
func NewOutboxEvent(aggregateType, aggregateID, eventType string, payload any) (*OutboxEvent, error) {
	if aggregateType == "" || aggregateID == "" || eventType == "" || payload == nil {
		return nil, ErrInvalidOutboxEvent
	}

	var rawPayload []byte
	switch p := payload.(type) {
	case []byte:
		if !json.Valid(p) {
			return nil, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidOutboxEvent)
		}
		rawPayload = p
	case json.RawMessage:
		if !json.Valid(p) {
			return nil, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidOutboxEvent)
		}
		rawPayload = p
	default:
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outbox event payload: %w", err)
		}
		rawPayload = encoded
	}

	return &OutboxEvent{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       json.RawMessage(rawPayload),
		Status:        OutboxStatusPending,
		RetryCount:    0,
		CreatedAt:     time.Now().UTC(),
		ProcessedAt:   nil,
		ErrorMessage:  nil,
	}, nil
}

// MarkPublished transitions status to PUBLISHED with a processed timestamp.
func (e *OutboxEvent) MarkPublished(now time.Time) {
	utc := now.UTC()
	e.Status = OutboxStatusPublished
	e.ProcessedAt = &utc
	e.ErrorMessage = nil
}

// MarkFailed records a failure, increments retry count, and transitions to FAILED if max retries reached.
func (e *OutboxEvent) MarkFailed(errMsg string, now time.Time, maxRetries int) {
	utc := now.UTC()
	e.RetryCount++
	e.ProcessedAt = &utc
	e.ErrorMessage = &errMsg
	if e.RetryCount >= maxRetries {
		e.Status = OutboxStatusFailed
	} else {
		e.Status = OutboxStatusPending
	}
}

// NewStockReservedCloudEvent builds a standard CloudEvent for stock reservation.
func NewStockReservedCloudEvent(
	reservationID, orderID, skuCode, warehouseID string,
	quantity, availableQtyAfter int,
	expiresAt time.Time,
	traceparent string,
) (*CloudEvent[StockReservedEventData], error) {
	if reservationID == "" || orderID == "" || skuCode == "" || warehouseID == "" || quantity <= 0 {
		return nil, ErrInvalidOutboxEvent
	}
	if traceparent == "" {
		traceparent = DefaultTraceparent
	}
	now := time.Now().UTC()
	return &CloudEvent[StockReservedEventData]{
		SpecVersion:     CloudEventsSpecVersion,
		ID:              uuid.New().String(),
		Source:          EventSourceInventoryService,
		Type:            EventTypeStockReserved,
		Subject:         fmt.Sprintf("sku:%s", skuCode),
		Time:            now.Format(time.RFC3339),
		DataContentType: DefaultContentTypeJSON,
		Traceparent:     traceparent,
		Data: StockReservedEventData{
			ReservationID:     reservationID,
			OrderID:           orderID,
			SKUCode:           skuCode,
			Quantity:          quantity,
			WarehouseID:       warehouseID,
			AvailableQtyAfter: availableQtyAfter,
			ExpiresAt:         expiresAt.UTC().Format(time.RFC3339),
		},
	}, nil
}

// NewStockReleasedCloudEvent builds a standard CloudEvent for stock release.
func NewStockReleasedCloudEvent(
	reservationID, orderID, skuCode, warehouseID string,
	quantityReleased, availableQtyAfter int,
	releaseReason string,
	traceparent string,
) (*CloudEvent[StockReleasedEventData], error) {
	if reservationID == "" || orderID == "" || skuCode == "" || warehouseID == "" || quantityReleased <= 0 {
		return nil, ErrInvalidOutboxEvent
	}
	if traceparent == "" {
		traceparent = DefaultTraceparent
	}
	if releaseReason == "" {
		releaseReason = "ORDER_CANCELLED_OR_EXPIRED"
	}
	now := time.Now().UTC()
	return &CloudEvent[StockReleasedEventData]{
		SpecVersion:     CloudEventsSpecVersion,
		ID:              uuid.New().String(),
		Source:          EventSourceInventoryService,
		Type:            EventTypeStockReleased,
		Subject:         fmt.Sprintf("sku:%s", skuCode),
		Time:            now.Format(time.RFC3339),
		DataContentType: DefaultContentTypeJSON,
		Traceparent:     traceparent,
		Data: StockReleasedEventData{
			ReservationID:     reservationID,
			OrderID:           orderID,
			SKUCode:           skuCode,
			QuantityReleased:  quantityReleased,
			WarehouseID:       warehouseID,
			AvailableQtyAfter: availableQtyAfter,
			ReleaseReason:     releaseReason,
		},
	}, nil
}

// NewExpiryWarningCloudEvent builds a standard CloudEvent for batch near expiry warning.
func NewExpiryWarningCloudEvent(
	batchCode, skuCode, warehouseID string,
	remainingQty, daysUntilExpiry int,
	bestBefore time.Time,
	recommendedAction string,
	traceparent string,
) (*CloudEvent[ExpiryWarningEventData], error) {
	if batchCode == "" || skuCode == "" || warehouseID == "" || remainingQty <= 0 {
		return nil, ErrInvalidOutboxEvent
	}
	if traceparent == "" {
		traceparent = DefaultTraceparent
	}
	if recommendedAction == "" {
		recommendedAction = "FLASH_SALE_PROMOTION"
	}
	now := time.Now().UTC()
	return &CloudEvent[ExpiryWarningEventData]{
		SpecVersion:     CloudEventsSpecVersion,
		ID:              uuid.New().String(),
		Source:          EventSourceInventoryService,
		Type:            EventTypeExpiryWarning,
		Subject:         fmt.Sprintf("batch:%s", batchCode),
		Time:            now.Format(time.RFC3339),
		DataContentType: DefaultContentTypeJSON,
		Traceparent:     traceparent,
		Data: ExpiryWarningEventData{
			BatchCode:         batchCode,
			SKUCode:           skuCode,
			WarehouseID:       warehouseID,
			RemainingQuantity: remainingQty,
			BestBefore:        bestBefore.UTC().Format(time.RFC3339),
			DaysUntilExpiry:   daysUntilExpiry,
			RecommendedAction: recommendedAction,
			DetectedAt:        now.Format(time.RFC3339),
		},
	}, nil
}

// NewStockCommittedCloudEvent builds a standard CloudEvent for finalized stock deduction.
func NewStockCommittedCloudEvent(
	reservationID, orderID, warehouseID string,
	items []StockCommittedItem,
	traceparent string,
) (*CloudEvent[StockCommittedEventData], error) {
	if reservationID == "" || orderID == "" || len(items) == 0 {
		return nil, ErrInvalidOutboxEvent
	}
	for _, itm := range items {
		if itm.SKUCode == "" || itm.Quantity <= 0 {
			return nil, ErrInvalidOutboxEvent
		}
	}
	if traceparent == "" {
		traceparent = DefaultTraceparent
	}
	now := time.Now().UTC()
	return &CloudEvent[StockCommittedEventData]{
		SpecVersion:     CloudEventsSpecVersion,
		ID:              uuid.New().String(),
		Source:          EventSourceInventoryService,
		Type:            EventTypeStockCommitted,
		Subject:         fmt.Sprintf("order:%s", orderID),
		Time:            now.Format(time.RFC3339),
		DataContentType: DefaultContentTypeJSON,
		Traceparent:     traceparent,
		Data: StockCommittedEventData{
			ReservationID: reservationID,
			OrderID:       orderID,
			WarehouseID:   warehouseID,
			Items:         items,
		},
	}, nil
}

