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

func (r *Repository) ProgressInternalDelivery(ctx context.Context, c domain.ProgressInternalDelivery) (*domain.InternalDeliveryProgressReceipt, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	s, err := scanShipment(tx.QueryRow(ctx, selectShipment+` WHERE id=$1 FOR UPDATE`, c.ShipmentID))
	if err != nil {
		return nil, err
	}
	var receipt domain.InternalDeliveryProgressReceipt
	var hash string
	err = tx.QueryRow(ctx, `SELECT shipment_id,actor_id,assignment_version,revision,status,id,occurred_at,request_hash FROM internal_delivery_progress_events WHERE actor_id=$1 AND idempotency_key=$2`, c.ActorID, c.IdempotencyKey).Scan(&receipt.ShipmentID, &receipt.ActorID, &receipt.AssignmentVersion, &receipt.Revision, &receipt.Status, &receipt.EventID, &receipt.OccurredAt, &hash)
	if err == nil {
		if hash != c.RequestHash() {
			return nil, domain.ErrIdempotencyConflict
		}
		return &receipt, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if s.CarrierCode != "INTERNAL" || s.CODAmountVND != 0 {
		return nil, fmt.Errorf("%w: prepaid INTERNAL shipment required", domain.ErrInvalidShipment)
	}
	var driver uuid.UUID
	var version int64
	err = tx.QueryRow(ctx, `SELECT driver_id,version FROM internal_deliveries WHERE shipment_id=$1`, s.ID).Scan(&driver, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if driver != c.ActorID {
		return nil, domain.ErrDriverNotAssigned
	}
	if version != c.AssignmentVersion || s.Revision != c.ExpectedRevision {
		return nil, domain.ErrConcurrentUpdate
	}
	var accepted bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM internal_delivery_acceptances WHERE shipment_id=$1 AND assignment_version=$2 AND driver_id=$3)`, s.ID, version, driver).Scan(&accepted); err != nil {
		return nil, err
	}
	if !accepted {
		return nil, fmt.Errorf("%w: current assignment must be accepted", domain.ErrInvalidTransition)
	}
	if err = domain.ValidateInternalProgress(s.Status, c.Status); err != nil {
		return nil, err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return nil, err
	}
	if err = s.Transition(c.Status, now); err != nil {
		return nil, err
	}
	s.LastEventAt = &now
	event := domain.NewDomainEvent(domain.InternalProgressEventName(c.Status), s, now)
	tracking := domain.TrackingEvent{ID: uuid.New(), ShipmentID: s.ID, ExternalEventID: event.ID.String(), CarrierCode: "INTERNAL", ProviderStatusCode: string(c.Status), MappedStatus: c.Status, MappingResult: "APPLIED", PublicDescription: "Company delivery status updated", InternalNote: "Employee " + driver.String() + ": " + c.Reason, EventAt: now, ReceivedAt: now}
	if err = saveShipmentTx(ctx, tx, s, c.ExpectedRevision, &tracking, &event); err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO internal_delivery_progress_events (id,shipment_id,actor_id,assignment_version,revision,status,reason,occurred_at,idempotency_key,request_hash) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, event.ID, s.ID, driver, version, s.Revision, s.Status, c.Reason, now, c.IdempotencyKey, c.RequestHash())
	if isUniqueViolation(err) {
		return nil, domain.ErrIdempotencyConflict
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &domain.InternalDeliveryProgressReceipt{ShipmentID: s.ID, ActorID: driver, AssignmentVersion: version, Revision: s.Revision, Status: s.Status, EventID: event.ID, OccurredAt: now}, nil
}
