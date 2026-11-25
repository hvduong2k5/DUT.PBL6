package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/shipping-service/internal/domain"
)

func (r *Repository) AcceptInternalDelivery(ctx context.Context, c domain.AcceptInternalDelivery) (*domain.InternalDeliveryAcceptance, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var code string
	var status domain.ShipmentStatus
	var cod int64
	err = tx.QueryRow(ctx, `SELECT carrier_code,status,cod_amount_vnd FROM shipments WHERE id=$1 FOR UPDATE`, c.ShipmentID).Scan(&code, &status, &cod)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	var result domain.InternalDeliveryAcceptance
	var hash string
	err = tx.QueryRow(ctx, `SELECT shipment_id,driver_id,assignment_version,accepted_at,request_hash FROM internal_delivery_acceptances WHERE driver_id=$1 AND idempotency_key=$2`, c.ActorID, c.IdempotencyKey).Scan(&result.ShipmentID, &result.DriverID, &result.AssignmentVersion, &result.AcceptedAt, &hash)
	if err == nil {
		if hash != c.RequestHash() {
			return nil, domain.ErrIdempotencyConflict
		}
		return &result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err = domain.ValidateInternalAssignmentState(code, status, cod); err != nil {
		return nil, err
	}
	var driver uuid.UUID
	var version int64
	err = tx.QueryRow(ctx, `SELECT driver_id,version FROM internal_deliveries WHERE shipment_id=$1`, c.ShipmentID).Scan(&driver, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if driver != c.ActorID {
		return nil, domain.ErrDriverNotAssigned
	}
	if version != c.ExpectedVersion {
		return nil, domain.ErrConcurrentUpdate
	}
	result = domain.InternalDeliveryAcceptance{ShipmentID: c.ShipmentID, DriverID: driver, AssignmentVersion: version}
	err = tx.QueryRow(ctx, `INSERT INTO internal_delivery_acceptances (shipment_id,driver_id,assignment_version,accepted_at,idempotency_key,request_hash) VALUES ($1,$2,$3,clock_timestamp(),$4,$5) RETURNING accepted_at`, c.ShipmentID, driver, version, c.IdempotencyKey, c.RequestHash()).Scan(&result.AcceptedAt)
	if isUniqueViolation(err) {
		return nil, domain.ErrIdempotencyConflict
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &result, nil
}
