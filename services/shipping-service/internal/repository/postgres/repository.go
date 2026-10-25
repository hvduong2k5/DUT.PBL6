package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/domain"
)

const shippingTopic = "shipping.events.v1"

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) BeginCreation(ctx context.Context, idempotencyKey string, shipment *domain.Shipment) (*domain.Shipment, bool, error) {
	recipient, err := json.Marshal(shipment.Recipient)
	if err != nil {
		return nil, false, err
	}
	parcel, err := json.Marshal(shipment.Parcel)
	if err != nil {
		return nil, false, err
	}
	commandTag, err := r.pool.Exec(ctx, `
		INSERT INTO shipments (
			id, shipment_number, idempotency_key, order_id, package_id, seal_code,
			carrier_code, service_code, status, recipient, parcel, cod_amount_vnd,
			checkout_fee_vnd, created_at, updated_at, revision
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		shipment.ID, shipment.ShipmentNumber, idempotencyKey, shipment.OrderID, shipment.PackageID,
		shipment.SealCode, shipment.CarrierCode, shipment.ServiceCode, shipment.Status, recipient, parcel,
		shipment.CODAmountVND, shipment.CheckoutFeeVND, shipment.CreatedAt, shipment.UpdatedAt, shipment.Revision)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, false, fmt.Errorf("%w: order/package already has a shipment", domain.ErrInvalidShipment)
		}
		return nil, false, err
	}
	if commandTag.RowsAffected() == 1 {
		return shipment, true, nil
	}
	existing, err := r.getByIdempotencyKey(ctx, idempotencyKey)
	return existing, false, err
}

func (r *Repository) Save(ctx context.Context, shipment *domain.Shipment, expectedRevision int64, tracking *domain.TrackingEvent, event *domain.DomainEvent) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if tracking != nil {
		tag, insertErr := tx.Exec(ctx, `
			INSERT INTO tracking_events (
				id, shipment_id, external_event_id, carrier_code, provider_status_code,
				provider_status_label, mapped_status, mapping_result, location,
				public_description, internal_note, event_at, received_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			ON CONFLICT (carrier_code, external_event_id) DO NOTHING`,
			tracking.ID, tracking.ShipmentID, tracking.ExternalEventID, tracking.CarrierCode,
			tracking.ProviderStatusCode, tracking.ProviderStatusLabel, tracking.MappedStatus,
			tracking.MappingResult, tracking.Location, tracking.PublicDescription,
			tracking.InternalNote, tracking.EventAt, tracking.ReceivedAt)
		if insertErr != nil {
			return insertErr
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrDuplicateEvent
		}
	}

	if shipment.Revision != expectedRevision {
		tag, updateErr := tx.Exec(ctx, `
			UPDATE shipments SET
				carrier_order_id=$2, tracking_code=$3, label_reference=$4, status=$5,
				provider_estimated_fee_vnd=$6, actual_fee_vnd=$7, failure_code=$8,
				failure_reason=$9, estimated_pickup_at=$10, estimated_delivery_at=$11,
				handed_over_at=$12, delivered_at=$13, last_event_at=$14, updated_at=$15, revision=$16
			WHERE id=$1 AND revision=$17`,
			shipment.ID, shipment.CarrierOrderID, shipment.TrackingCode, shipment.LabelReference,
			shipment.Status, shipment.ProviderEstimatedFeeVND, shipment.ActualFeeVND,
			shipment.FailureCode, shipment.FailureReason, shipment.EstimatedPickupAt,
			shipment.EstimatedDeliveryAt, shipment.HandedOverAt, shipment.DeliveredAt,
			shipment.LastEventAt, shipment.UpdatedAt, shipment.Revision, expectedRevision)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrConcurrentUpdate
		}
	}

	if event != nil {
		payload, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			return marshalErr
		}
		_, insertErr := tx.Exec(ctx, `
			INSERT INTO outbox_events (id, aggregate_id, aggregate_type, event_type, topic, partition_key, payload, occurred_at)
			VALUES ($1,$2,'shipment',$3,$4,$5,$6,$7)`,
			event.ID, event.AggregateID, event.Type, shippingTopic, event.AggregateID.String(), payload, event.OccurredAt)
		if insertErr != nil {
			return insertErr
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	return scanShipment(r.pool.QueryRow(ctx, selectShipment+` WHERE id=$1`, id))
}

func (r *Repository) GetByTracking(ctx context.Context, carrierCode, trackingCode string) (*domain.Shipment, error) {
	s, err := scanShipment(r.pool.QueryRow(ctx, selectShipment+` WHERE carrier_code=$1 AND tracking_code=$2`, carrierCode, trackingCode))
	if errors.Is(err, domain.ErrShipmentNotFound) {
		return nil, domain.ErrTrackingNotFound
	}
	return s, err
}

func (r *Repository) getByIdempotencyKey(ctx context.Context, key string) (*domain.Shipment, error) {
	return scanShipment(r.pool.QueryRow(ctx, selectShipment+` WHERE idempotency_key=$1`, key))
}

func (r *Repository) ListByOrder(ctx context.Context, orderID string) ([]domain.Shipment, error) {
	rows, err := r.pool.Query(ctx, selectShipment+` WHERE order_id=$1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	return scanShipments(rows)
}

func (r *Repository) List(ctx context.Context, filter application.ListFilter) ([]domain.Shipment, error) {
	rows, err := r.pool.Query(ctx, selectShipment+`
		WHERE ($1='' OR status=$1)
		  AND ($2='' OR carrier_code=$2)
		  AND ($3='' OR order_id=$3)
		ORDER BY updated_at DESC LIMIT $4 OFFSET $5`,
		filter.Status, filter.CarrierCode, filter.OrderID, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	return scanShipments(rows)
}

func (r *Repository) TrackingTimeline(ctx context.Context, shipmentID uuid.UUID) ([]domain.TrackingEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, shipment_id, external_event_id, carrier_code, provider_status_code,
		       provider_status_label, mapped_status, mapping_result, location,
		       public_description, internal_note, event_at, received_at
		FROM tracking_events WHERE shipment_id=$1 ORDER BY event_at, received_at`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.TrackingEvent
	for rows.Next() {
		var event domain.TrackingEvent
		if err := rows.Scan(&event.ID, &event.ShipmentID, &event.ExternalEventID, &event.CarrierCode,
			&event.ProviderStatusCode, &event.ProviderStatusLabel, &event.MappedStatus,
			&event.MappingResult, &event.Location, &event.PublicDescription, &event.InternalNote,
			&event.EventAt, &event.ReceivedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

const selectShipment = `SELECT
	id, shipment_number, order_id, package_id, seal_code, carrier_code, service_code,
	carrier_order_id, tracking_code, label_reference, status, recipient, parcel,
	cod_amount_vnd, checkout_fee_vnd, provider_estimated_fee_vnd, actual_fee_vnd,
	failure_code, failure_reason, estimated_pickup_at, estimated_delivery_at,
	handed_over_at, delivered_at, last_event_at, created_at, updated_at, revision
FROM shipments`

type rowScanner interface{ Scan(...any) error }

func scanShipment(row rowScanner) (*domain.Shipment, error) {
	var shipment domain.Shipment
	var recipient, parcel []byte
	err := row.Scan(&shipment.ID, &shipment.ShipmentNumber, &shipment.OrderID, &shipment.PackageID,
		&shipment.SealCode, &shipment.CarrierCode, &shipment.ServiceCode, &shipment.CarrierOrderID,
		&shipment.TrackingCode, &shipment.LabelReference, &shipment.Status, &recipient, &parcel,
		&shipment.CODAmountVND, &shipment.CheckoutFeeVND, &shipment.ProviderEstimatedFeeVND,
		&shipment.ActualFeeVND, &shipment.FailureCode, &shipment.FailureReason,
		&shipment.EstimatedPickupAt, &shipment.EstimatedDeliveryAt, &shipment.HandedOverAt,
		&shipment.DeliveredAt, &shipment.LastEventAt, &shipment.CreatedAt, &shipment.UpdatedAt,
		&shipment.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(recipient, &shipment.Recipient); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(parcel, &shipment.Parcel); err != nil {
		return nil, err
	}
	return &shipment, nil
}

func scanShipments(rows pgx.Rows) ([]domain.Shipment, error) {
	defer rows.Close()
	var result []domain.Shipment
	for rows.Next() {
		shipment, err := scanShipment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *shipment)
	}
	return result, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
