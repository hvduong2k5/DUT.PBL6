package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/shipping-service/internal/domain"
)

const driverSummarySelect = `SELECT s.id,s.shipment_number,s.order_id,s.package_id,s.status,s.revision,d.version,
 EXISTS(SELECT 1 FROM internal_delivery_acceptances a WHERE a.shipment_id=s.id AND a.assignment_version=d.version AND a.driver_id=d.driver_id)
 FROM shipments s JOIN internal_deliveries d ON d.shipment_id=s.id
 WHERE d.driver_id=$1 AND s.carrier_code='INTERNAL' AND s.cod_amount_vnd=0`

func (r *Repository) ListDriverShipments(ctx context.Context, actor uuid.UUID, limit, offset int) (*domain.DriverShipmentList, error) {
	// Repeatable snapshot binds total/items to the same current ownership view.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	result := domain.DriverShipmentList{Items: []domain.DriverShipmentSummary{}, Limit: limit, Offset: offset}
	err = tx.QueryRow(ctx, `SELECT count(*) FROM shipments s JOIN internal_deliveries d ON d.shipment_id=s.id WHERE d.driver_id=$1 AND s.carrier_code='INTERNAL' AND s.cod_amount_vnd=0`, actor).Scan(&result.Total)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, driverSummarySelect+` ORDER BY d.assigned_at DESC,s.id LIMIT $2 OFFSET $3`, actor, limit, offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item domain.DriverShipmentSummary
		if err = rows.Scan(&item.ShipmentID, &item.ShipmentNumber, &item.OrderID, &item.PackageID, &item.Status, &item.Revision, &item.AssignmentVersion, &item.Accepted); err != nil {
			rows.Close()
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Repository) GetDriverShipment(ctx context.Context, actor, id uuid.UUID) (*domain.DriverShipmentDetail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var result domain.DriverShipmentDetail
	err = tx.QueryRow(ctx, driverSummarySelect+` AND s.id=$2`, actor, id).Scan(&result.ShipmentID, &result.ShipmentNumber, &result.OrderID, &result.PackageID, &result.Status, &result.Revision, &result.AssignmentVersion, &result.Accepted)
	// Missing and unowned are indistinguishable: no recipient/existence oracle.
	if err == pgx.ErrNoRows {
		return nil, domain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	s, err := scanShipment(tx.QueryRow(ctx, selectShipment+` WHERE id=$1`, id))
	if err != nil {
		return nil, err
	}
	result.Recipient = s.Recipient
	result.Parcel = s.Parcel
	result.TrackingCode = s.TrackingCode
	result.HandedOverAt = s.HandedOverAt
	result.EstimatedDeliveryAt = s.EstimatedDeliveryAt
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &result, nil
}
