package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
)

type AddressRepository struct {
	pool *pgxpool.Pool
}

func NewAddressRepository(pool *pgxpool.Pool) *AddressRepository {
	return &AddressRepository{pool: pool}
}

// CreateAddress inserts a new shipping address.
func (r *AddressRepository) CreateAddress(ctx context.Context, addr *domain.ShippingAddress) error {
	query := `
		INSERT INTO shipping_addresses (
			id, customer_id, recipient_name, phone_number, street_address,
			ward_code, ward_name, province_code, province_name,
			latitude, longitude, label, is_default, is_deleted, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`
	_, err := r.pool.Exec(ctx, query,
		addr.ID, addr.CustomerID, addr.RecipientName, addr.PhoneNumber, addr.StreetAddress,
		addr.WardCode, addr.WardName, addr.ProvinceCode, addr.ProvinceName,
		addr.Latitude, addr.Longitude, addr.Label, addr.IsDefault, addr.IsDeleted, addr.CreatedAt, addr.UpdatedAt,
	)
	return err
}

// SwitchDefaultAddress atomically switches default shipping address for a customer.
// Uses SELECT ... FOR UPDATE to serialize concurrent attempts and leverages PostgreSQL
// Partial Unique Index as the ultimate integrity guarantee.
func (r *AddressRepository) SwitchDefaultAddress(ctx context.Context, customerID, newDefaultAddressID uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Step 1: Lock all addresses of this customer to strictly serialize concurrent switch requests
	var targetExists bool
	rows, err := tx.Query(ctx,
		`SELECT id FROM shipping_addresses WHERE customer_id = $1 AND is_deleted = FALSE FOR UPDATE`,
		customerID,
	)
	if err != nil {
		return fmt.Errorf("failed locking customer addresses: %w", err)
	}

	for rows.Next() {
		var addrID uuid.UUID
		if scanErr := rows.Scan(&addrID); scanErr == nil {
			if addrID == newDefaultAddressID {
				targetExists = true
			}
		}
	}
	rows.Close()

	if !targetExists {
		return domain.ErrAddressNotFound
	}

	// Step 2: Demote previous default address
	_, err = tx.Exec(ctx,
		`UPDATE shipping_addresses 
		 SET is_default = FALSE, updated_at = CURRENT_TIMESTAMP 
		 WHERE customer_id = $1 AND is_default = TRUE AND is_deleted = FALSE`,
		customerID,
	)
	if err != nil {
		return fmt.Errorf("failed resetting existing default address: %w", err)
	}

	// Step 3: Promote target address to default
	_, err = tx.Exec(ctx,
		`UPDATE shipping_addresses 
		 SET is_default = TRUE, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = $1 AND customer_id = $2`,
		newDefaultAddressID, customerID,
	)
	if err != nil {
		return fmt.Errorf("failed promoting new default address: %w", err)
	}

	// Step 4: Write DefaultAddressSwitchedEvent into outbox table in the same transaction
	outboxPayload, err := json.Marshal(map[string]any{
		"customer_id":        customerID.String(),
		"default_address_id": newDefaultAddressID.String(),
		"switched_at":        time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("failed marshaling outbox event: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, topic) 
		 VALUES ($1, 'Address', $2, 'DefaultAddressSwitchedEvent', $3, 'profile.events.v1')`,
		uuid.New(), customerID.String(), outboxPayload,
	)
	if err != nil {
		return fmt.Errorf("failed inserting outbox event: %w", err)
	}

	return tx.Commit(ctx)
}

// CountDefaultAddresses counts how many active default addresses exist for a customer.
func (r *AddressRepository) CountDefaultAddresses(ctx context.Context, customerID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM shipping_addresses 
		WHERE customer_id = $1 AND is_default = TRUE AND is_deleted = FALSE
	`
	err := r.pool.QueryRow(ctx, query, customerID).Scan(&count)
	return count, err
}

// GetAddressByID fetches a specific active address by ID.
func (r *AddressRepository) GetAddressByID(ctx context.Context, id uuid.UUID) (*domain.ShippingAddress, error) {
	query := `
		SELECT id, customer_id, recipient_name, phone_number, street_address,
		       ward_code, ward_name, province_code, province_name,
		       latitude, longitude, label, is_default, is_deleted, created_at, updated_at
		FROM shipping_addresses
		WHERE id = $1 AND is_deleted = FALSE
	`
	var a domain.ShippingAddress
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
		&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
		&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		return nil, err
	}
	return &a, nil
}

// GetDefaultAddress fetches the default active address for a customer (Checkout Critical Path O(1)).
func (r *AddressRepository) GetDefaultAddress(ctx context.Context, customerID uuid.UUID) (*domain.ShippingAddress, error) {
	query := `
		SELECT id, customer_id, recipient_name, phone_number, street_address,
		       ward_code, ward_name, province_code, province_name,
		       latitude, longitude, label, is_default, is_deleted, created_at, updated_at
		FROM shipping_addresses
		WHERE customer_id = $1 AND is_default = TRUE AND is_deleted = FALSE
		LIMIT 1
	`
	var a domain.ShippingAddress
	err := r.pool.QueryRow(ctx, query, customerID).Scan(
		&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
		&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
		&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		return nil, err
	}
	return &a, nil
}

// ListAddressesByCustomerID returns all non-deleted addresses for a customer.
func (r *AddressRepository) ListAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.ShippingAddress, error) {
	query := `
		SELECT id, customer_id, recipient_name, phone_number, street_address,
		       ward_code, ward_name, province_code, province_name,
		       latitude, longitude, label, is_default, is_deleted, created_at, updated_at
		FROM shipping_addresses
		WHERE customer_id = $1 AND is_deleted = FALSE
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []*domain.ShippingAddress
	for rows.Next() {
		var a domain.ShippingAddress
		if err := rows.Scan(
			&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
			&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
			&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		addresses = append(addresses, &a)
	}
	return addresses, rows.Err()
}

// DeleteAddress performs soft delete on a customer's address.
func (r *AddressRepository) DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	query := `
		UPDATE shipping_addresses
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND is_deleted = FALSE
	`
	tag, err := r.pool.Exec(ctx, query, addressID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}
