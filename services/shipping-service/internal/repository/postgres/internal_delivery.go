package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/shipping-service/internal/domain"
)

func (r *Repository) AssignInternalDelivery(ctx context.Context, c domain.AssignInternalDelivery) (*domain.InternalDeliveryAssignment, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Serialize with shipment transitions and all assignments for this resource.
	var carrierCode string
	var state domain.ShipmentStatus
	var cod int64
	err = tx.QueryRow(ctx, `SELECT carrier_code,status,cod_amount_vnd FROM shipments WHERE id=$1 FOR UPDATE`, c.ShipmentID).Scan(&carrierCode, &state, &cod)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	var savedHash string
	replay := domain.InternalDeliveryAssignment{}
	err = tx.QueryRow(ctx, `SELECT shipment_id,driver_id,assigned_by,version,reason,assigned_at,request_hash
		FROM internal_delivery_assignment_events WHERE assigned_by=$1 AND idempotency_key=$2`, c.ActorID, c.IdempotencyKey).
		Scan(&replay.ShipmentID, &replay.DriverID, &replay.ActorID, &replay.Version, &replay.Reason, &replay.AssignedAt, &savedHash)
	if err == nil {
		if savedHash != c.RequestHash() {
			return nil, domain.ErrIdempotencyConflict
		}
		return &replay, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err := domain.ValidateInternalAssignmentState(carrierCode, state, cod); err != nil {
		return nil, err
	}
	var version int64
	var previous uuid.UUID
	err = tx.QueryRow(ctx, `SELECT driver_id,version FROM internal_deliveries WHERE shipment_id=$1`, c.ShipmentID).Scan(&previous, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if version != c.ExpectedVersion {
		return nil, domain.ErrConcurrentUpdate
	}
	if previous == c.DriverID {
		return nil, fmt.Errorf("%w: driver is already assigned", domain.ErrInvalidShipment)
	}
	var assignedAt time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&assignedAt); err != nil {
		return nil, err
	}
	result := domain.InternalDeliveryAssignment{ShipmentID: c.ShipmentID, DriverID: c.DriverID, ActorID: c.ActorID, Version: version + 1, Reason: c.Reason, AssignedAt: assignedAt}
	var previousID *uuid.UUID
	if previous != uuid.Nil {
		previousID = &previous
	}
	_, err = tx.Exec(ctx, `INSERT INTO internal_delivery_assignment_events
		(id,shipment_id,driver_id,previous_driver_id,assigned_by,version,reason,assigned_at,idempotency_key,request_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, uuid.New(), result.ShipmentID, result.DriverID, previousID, result.ActorID, result.Version, result.Reason, result.AssignedAt, c.IdempotencyKey, c.RequestHash())
	if isUniqueViolation(err) {
		return nil, domain.ErrIdempotencyConflict
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO internal_deliveries (shipment_id,driver_id,assigned_by,version,reason,assigned_at)
		VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (shipment_id) DO UPDATE SET
		driver_id=EXCLUDED.driver_id,assigned_by=EXCLUDED.assigned_by,version=EXCLUDED.version,reason=EXCLUDED.reason,assigned_at=EXCLUDED.assigned_at`,
		result.ShipmentID, result.DriverID, result.ActorID, result.Version, result.Reason, result.AssignedAt)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Repository) GetInternalDeliveryAssignment(ctx context.Context, id uuid.UUID) (*domain.InternalDeliveryAssignment, error) {
	result := domain.InternalDeliveryAssignment{}
	err := r.pool.QueryRow(ctx, `SELECT shipment_id,driver_id,assigned_by,version,reason,assigned_at FROM internal_deliveries WHERE shipment_id=$1`, id).
		Scan(&result.ShipmentID, &result.DriverID, &result.ActorID, &result.Version, &result.Reason, &result.AssignedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAssignmentNotFound
	}
	return &result, err
}

func (r *Repository) InternalDeliveryAssignmentHistory(ctx context.Context, id uuid.UUID) ([]domain.InternalDeliveryAssignmentEvent, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,shipment_id,driver_id,previous_driver_id,assigned_by,version,reason,assigned_at
		FROM internal_delivery_assignment_events WHERE shipment_id=$1 ORDER BY version`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.InternalDeliveryAssignmentEvent{}
	for rows.Next() {
		var event domain.InternalDeliveryAssignmentEvent
		if err := rows.Scan(&event.ID, &event.ShipmentID, &event.DriverID, &event.PreviousDriverID, &event.ActorID, &event.Version, &event.Reason, &event.AssignedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
