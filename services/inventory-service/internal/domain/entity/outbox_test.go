package entity_test

import (
	"encoding/json"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutboxEvent_Valid(t *testing.T) {
	payload := map[string]string{"foo": "bar"}
	event, err := entity.NewOutboxEvent("StockReservation", "res-123", entity.EventTypeStockReserved, payload)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, "StockReservation", event.AggregateType)
	assert.Equal(t, "res-123", event.AggregateID)
	assert.Equal(t, entity.EventTypeStockReserved, event.EventType)
	assert.Equal(t, entity.OutboxStatusPending, event.Status)
	assert.Equal(t, 0, event.RetryCount)
	assert.Nil(t, event.ProcessedAt)
	assert.Nil(t, event.ErrorMessage)
	assert.JSONEq(t, `{"foo":"bar"}`, string(event.Payload))
}

func TestNewOutboxEvent_ValidRawJSON(t *testing.T) {
	raw := []byte(`{"key":"value"}`)
	event, err := entity.NewOutboxEvent("Batch", "batch-123", entity.EventTypeExpiryWarning, raw)
	require.NoError(t, err)
	assert.Equal(t, raw, []byte(event.Payload))
}

func TestNewOutboxEvent_Invalid(t *testing.T) {
	// Empty aggregate type
	_, err := entity.NewOutboxEvent("", "id", "type", map[string]string{})
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Empty aggregate ID
	_, err = entity.NewOutboxEvent("agg", "", "type", map[string]string{})
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Empty event type
	_, err = entity.NewOutboxEvent("agg", "id", "", map[string]string{})
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Nil payload
	_, err = entity.NewOutboxEvent("agg", "id", "type", nil)
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Invalid JSON bytes
	_, err = entity.NewOutboxEvent("agg", "id", "type", []byte("invalid json"))
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)
}

func TestOutboxEvent_MarkPublished(t *testing.T) {
	event, err := entity.NewOutboxEvent("StockReservation", "res-123", entity.EventTypeStockReserved, map[string]string{})
	require.NoError(t, err)

	now := time.Now().UTC()
	event.MarkPublished(now)

	assert.Equal(t, entity.OutboxStatusPublished, event.Status)
	require.NotNil(t, event.ProcessedAt)
	assert.Equal(t, now, *event.ProcessedAt)
	assert.Nil(t, event.ErrorMessage)
}

func TestOutboxEvent_MarkFailed(t *testing.T) {
	event, err := entity.NewOutboxEvent("StockReservation", "res-123", entity.EventTypeStockReserved, map[string]string{})
	require.NoError(t, err)

	now := time.Now().UTC()
	// Retry 1: still pending
	event.MarkFailed("connection refused", now, 3)
	assert.Equal(t, entity.OutboxStatusPending, event.Status)
	assert.Equal(t, 1, event.RetryCount)
	require.NotNil(t, event.ErrorMessage)
	assert.Equal(t, "connection refused", *event.ErrorMessage)

	// Retry 2: still pending
	event.MarkFailed("timeout", now, 3)
	assert.Equal(t, entity.OutboxStatusPending, event.Status)
	assert.Equal(t, 2, event.RetryCount)

	// Retry 3: max retries reached, transitions to FAILED
	event.MarkFailed("dead letter", now, 3)
	assert.Equal(t, entity.OutboxStatusFailed, event.Status)
	assert.Equal(t, 3, event.RetryCount)
	assert.Equal(t, "dead letter", *event.ErrorMessage)
}

func TestNewStockReservedCloudEvent(t *testing.T) {
	expiresAt := time.Now().Add(15 * time.Minute).UTC()
	ce, err := entity.NewStockReservedCloudEvent(
		"res-001",
		"ord-001",
		"MX-GION-500G",
		"KHO-HUONG-THUY",
		10,
		90,
		expiresAt,
		"trace-001",
	)
	require.NoError(t, err)
	assert.Equal(t, entity.CloudEventsSpecVersion, ce.SpecVersion)
	assert.Equal(t, entity.EventTypeStockReserved, ce.Type)
	assert.Equal(t, "sku:MX-GION-500G", ce.Subject)
	assert.Equal(t, "trace-001", ce.Traceparent)
	assert.Equal(t, 10, ce.Data.Quantity)
	assert.Equal(t, 90, ce.Data.AvailableQtyAfter)

	// Validate JSON marshaling
	bytes, err := json.Marshal(ce)
	require.NoError(t, err)
	assert.Contains(t, string(bytes), "vn.omama.inventory.stock.reserved.v1")

	// Invalid input
	_, err = entity.NewStockReservedCloudEvent("", "ord-1", "sku-1", "wh-1", 10, 90, expiresAt, "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)
}

func TestNewStockReleasedCloudEvent(t *testing.T) {
	ce, err := entity.NewStockReleasedCloudEvent(
		"res-001",
		"ord-001",
		"MX-GION-500G",
		"KHO-HUONG-THUY",
		5,
		95,
		"ORDER_TIMEOUT",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, entity.EventTypeStockReleased, ce.Type)
	assert.Equal(t, "ORDER_TIMEOUT", ce.Data.ReleaseReason)
	assert.Equal(t, entity.DefaultTraceparent, ce.Traceparent)

	// Invalid input
	_, err = entity.NewStockReleasedCloudEvent("res-1", "ord-1", "sku", "wh", 0, 95, "", "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)
}

func TestNewExpiryWarningCloudEvent(t *testing.T) {
	bestBefore := time.Now().Add(30 * 24 * time.Hour).UTC()
	ce, err := entity.NewExpiryWarningCloudEvent(
		"LOT-2026-001",
		"MX-GION-500G",
		"KHO-HUONG-THUY",
		50,
		30,
		bestBefore,
		"FLASH_SALE_PROMOTION",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, entity.EventTypeExpiryWarning, ce.Type)
	assert.Equal(t, "batch:LOT-2026-001", ce.Subject)
	assert.Equal(t, 30, ce.Data.DaysUntilExpiry)
	assert.Equal(t, "FLASH_SALE_PROMOTION", ce.Data.RecommendedAction)

	// Invalid input
	_, err = entity.NewExpiryWarningCloudEvent("", "sku", "wh", 50, 30, bestBefore, "", "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)
}

func TestNewStockCommittedCloudEvent(t *testing.T) {
	items := []entity.StockCommittedItem{
		{SKUCode: "MX-GION-500G", Quantity: 5},
		{SKUCode: "MX-ME-DEN-250G", Quantity: 2},
	}
	ce, err := entity.NewStockCommittedCloudEvent(
		"res-001",
		"ord-001",
		"KHO-HUONG-THUY",
		items,
		"trace-001",
	)
	require.NoError(t, err)
	assert.Equal(t, entity.CloudEventsSpecVersion, ce.SpecVersion)
	assert.Equal(t, entity.EventTypeStockCommitted, ce.Type)
	assert.Equal(t, "order:ord-001", ce.Subject)
	assert.Equal(t, "trace-001", ce.Traceparent)
	assert.Equal(t, "ord-001", ce.Data.OrderID)
	assert.Equal(t, "res-001", ce.Data.ReservationID)
	assert.Equal(t, "KHO-HUONG-THUY", ce.Data.WarehouseID)
	assert.Equal(t, 2, len(ce.Data.Items))
	assert.Equal(t, "MX-GION-500G", ce.Data.Items[0].SKUCode)
	assert.Equal(t, 5, ce.Data.Items[0].Quantity)

	// Validate JSON marshaling
	bytes, err := json.Marshal(ce)
	require.NoError(t, err)
	assert.Contains(t, string(bytes), "vn.omama.inventory.stock.committed.v1")

	// Invalid input: empty items
	_, err = entity.NewStockCommittedCloudEvent("res-001", "ord-001", "wh", nil, "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Invalid input: empty orderID
	_, err = entity.NewStockCommittedCloudEvent("res-001", "", "wh", items, "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)

	// Invalid input: invalid item qty
	invalidItems := []entity.StockCommittedItem{{SKUCode: "SKU-1", Quantity: 0}}
	_, err = entity.NewStockCommittedCloudEvent("res-001", "ord-001", "wh", invalidItems, "")
	assert.ErrorIs(t, err, entity.ErrInvalidOutboxEvent)
}

