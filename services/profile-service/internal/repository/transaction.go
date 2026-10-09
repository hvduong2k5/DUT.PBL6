package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/profile-service/internal/domain"
)

func lockCustomer(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	var found uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM customer_profiles WHERE id=$1 FOR UPDATE`, id).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCustomerNotFound
	}
	return err
}
func insertEvent(ctx context.Context, tx pgx.Tx, aggregate, id, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,aggregate_type,aggregate_id,event_type,payload,topic)
 VALUES($1,$2,$3,$4,$5,'profile.events.v1')`, uuid.New(), aggregate, id, event, data)
	return err
}
func canonicalArea(ctx context.Context, tx pgx.Tx, a *domain.ShippingAddress) error {
	err := tx.QueryRow(ctx, `SELECT w.full_name,p.full_name FROM administrative_units w
 JOIN administrative_units p ON w.parent_code=p.code
 WHERE w.code=$1 AND p.code=$2 AND w.level='WARD' AND p.level='PROVINCE'`, a.WardCode, a.ProvinceCode).Scan(&a.WardName, &a.ProvinceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.ValidationError{Field: "ward_code", Message: "ward must belong to the selected province"}
	}
	return err
}
