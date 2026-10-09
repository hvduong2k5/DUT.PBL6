package repository

import (
	"context"
	"errors"

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

// All address mutations lock the stable customer row before touching child rows.
func (r *AddressRepository) CreateAddress(ctx context.Context, a *domain.ShippingAddress) error {
	if err := domain.ValidateAddress(a); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockCustomer(ctx, tx, a.CustomerID); err != nil {
		return err
	}
	if err = canonicalArea(ctx, tx, a); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM shipping_addresses WHERE customer_id=$1 AND NOT is_deleted`, a.CustomerID).Scan(&count); err != nil {
		return err
	}
	a.IsDefault = a.IsDefault || count == 0
	a.Version = 1
	a.IsDeleted = false
	if a.IsDefault {
		if _, err = tx.Exec(ctx, `UPDATE shipping_addresses SET is_default=false,version=version+1,updated_at=NOW() WHERE customer_id=$1 AND is_default AND NOT is_deleted`, a.CustomerID); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO shipping_addresses(id,customer_id,recipient_name,phone_number,street_address,ward_code,ward_name,province_code,province_name,latitude,longitude,label,is_default,is_deleted,version,created_at,updated_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,false,1,$14,$15)`, a.ID, a.CustomerID, a.RecipientName, a.PhoneNumber, a.StreetAddress, a.WardCode, a.WardName, a.ProvinceCode, a.ProvinceName, a.Latitude, a.Longitude, a.Label, a.IsDefault, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return err
	}
	if err = insertEvent(ctx, tx, "Address", a.CustomerID.String(), "vn.omama.profile.address_created.v1", map[string]any{"customer_id": a.CustomerID, "address_id": a.ID, "is_default": a.IsDefault}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *AddressRepository) UpdateAddress(ctx context.Context, a *domain.ShippingAddress) error {
	if err := domain.ValidateAddress(a); err != nil {
		return err
	}
	if a.Version < 1 {
		return &domain.ValidationError{Field: "version", Message: "must be positive"}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockCustomer(ctx, tx, a.CustomerID); err != nil {
		return err
	}
	if err = canonicalArea(ctx, tx, a); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE shipping_addresses SET recipient_name=$1,phone_number=$2,street_address=$3,ward_code=$4,ward_name=$5,province_code=$6,province_name=$7,latitude=$8,longitude=$9,label=$10,version=version+1,updated_at=NOW()
 WHERE id=$11 AND customer_id=$12 AND NOT is_deleted AND version=$13`, a.RecipientName, a.PhoneNumber, a.StreetAddress, a.WardCode, a.WardName, a.ProvinceCode, a.ProvinceName, a.Latitude, a.Longitude, a.Label, a.ID, a.CustomerID, a.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return addressConflict(ctx, tx, a.CustomerID, a.ID)
	}
	if err = tx.QueryRow(ctx, `SELECT version,is_default,created_at,updated_at FROM shipping_addresses WHERE id=$1 AND customer_id=$2`, a.ID, a.CustomerID).Scan(&a.Version, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return err
	}
	if err = insertEvent(ctx, tx, "Address", a.CustomerID.String(), "vn.omama.profile.address_updated.v1", map[string]any{"customer_id": a.CustomerID, "address_id": a.ID, "new_version": a.Version}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func addressConflict(ctx context.Context, tx pgx.Tx, customerID, addressID uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shipping_addresses WHERE id=$1 AND customer_id=$2 AND NOT is_deleted)`, addressID, customerID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrAddressNotFound
	}
	return domain.ErrOptimisticLockConflict
}
func (r *AddressRepository) SwitchDefaultAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	return r.switchDefault(ctx, customerID, addressID, 0)
}
func (r *AddressRepository) SwitchDefaultAddressWithVersion(ctx context.Context, customerID, addressID uuid.UUID, version int) error {
	return r.switchDefault(ctx, customerID, addressID, version)
}
func (r *AddressRepository) switchDefault(ctx context.Context, customerID, addressID uuid.UUID, version int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockCustomer(ctx, tx, customerID); err != nil {
		return err
	}
	var currentVersion int
	var isDefault bool
	err = tx.QueryRow(ctx, `SELECT version,is_default FROM shipping_addresses WHERE id=$1 AND customer_id=$2 AND NOT is_deleted`, addressID, customerID).Scan(&currentVersion, &isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAddressNotFound
	}
	if err != nil {
		return err
	}
	// An already-default target is a no-op, including retries of the same intent.
	if isDefault {
		return tx.Commit(ctx)
	}
	if version > 0 && version != currentVersion {
		return domain.ErrOptimisticLockConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE shipping_addresses SET is_default=false,version=version+1,updated_at=NOW() WHERE customer_id=$1 AND is_default AND NOT is_deleted`, customerID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE shipping_addresses SET is_default=true,version=version+1,updated_at=NOW() WHERE id=$1 AND customer_id=$2`, addressID, customerID); err != nil {
		return err
	}
	if err = insertEvent(ctx, tx, "Address", customerID.String(), "vn.omama.profile.default_address_switched.v1", map[string]any{"customer_id": customerID, "default_address_id": addressID}); err != nil {
		return err
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
	return r.getAddress(ctx, id, uuid.Nil)
}
func (r *AddressRepository) getAddress(ctx context.Context, id, customerID uuid.UUID) (*domain.ShippingAddress, error) {
	query := `
		SELECT id, customer_id, recipient_name, phone_number, street_address,
		       ward_code, ward_name, province_code, province_name,
		       latitude, longitude, label, is_default, is_deleted, version, created_at, updated_at
		FROM shipping_addresses
		WHERE id = $1 AND is_deleted = FALSE AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR customer_id=$2)
	`
	var a domain.ShippingAddress
	err := r.pool.QueryRow(ctx, query, id, customerID).Scan(
		&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
		&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
		&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.Version, &a.CreatedAt, &a.UpdatedAt,
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
		       latitude, longitude, label, is_default, is_deleted, version, created_at, updated_at
		FROM shipping_addresses
		WHERE customer_id = $1 AND is_default = TRUE AND is_deleted = FALSE
		LIMIT 1
	`
	var a domain.ShippingAddress
	err := r.pool.QueryRow(ctx, query, customerID).Scan(
		&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
		&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
		&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.Version, &a.CreatedAt, &a.UpdatedAt,
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
		       latitude, longitude, label, is_default, is_deleted, version, created_at, updated_at
		FROM shipping_addresses
		WHERE customer_id = $1 AND is_deleted = FALSE
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addresses := make([]*domain.ShippingAddress, 0)
	for rows.Next() {
		var a domain.ShippingAddress
		if err := rows.Scan(
			&a.ID, &a.CustomerID, &a.RecipientName, &a.PhoneNumber, &a.StreetAddress,
			&a.WardCode, &a.WardName, &a.ProvinceCode, &a.ProvinceName,
			&a.Latitude, &a.Longitude, &a.Label, &a.IsDefault, &a.IsDeleted, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		addresses = append(addresses, &a)
	}
	return addresses, rows.Err()
}

func (r *AddressRepository) DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	return r.DeleteAddressWithVersion(ctx, customerID, addressID, 0)
}
func (r *AddressRepository) DeleteAddressWithVersion(ctx context.Context, customerID, addressID uuid.UUID, version int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockCustomer(ctx, tx, customerID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE shipping_addresses SET is_deleted=true,is_default=false,version=version+1,updated_at=NOW()
	 WHERE id=$1 AND customer_id=$2 AND NOT is_deleted AND ($3=0 OR version=$3)
	 `, addressID, customerID, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return addressConflict(ctx, tx, customerID, addressID)
	}
	// Repair any missing default and select deterministically by latest update, then ID.
	var replacement uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM shipping_addresses WHERE customer_id=$1 AND NOT is_deleted ORDER BY is_default DESC,updated_at DESC,id ASC LIMIT 1`, customerID).Scan(&replacement)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		if _, err = tx.Exec(ctx, `UPDATE shipping_addresses SET is_default=true,version=version+1,updated_at=NOW() WHERE id=$1 AND NOT is_default`, replacement); err != nil {
			return err
		}
	}
	if err = insertEvent(ctx, tx, "Address", customerID.String(), "vn.omama.profile.address_deleted.v1", map[string]any{"customer_id": customerID, "address_id": addressID, "default_address_id": replacement}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type StreetWardMapping struct {
	ID                   uuid.UUID `json:"id"`
	StreetName           string    `json:"street_name"`
	StreetNameUnaccented string    `json:"street_name_unaccented"`
	WardCode             string    `json:"ward_code"`
	WardName             string    `json:"ward_name"`
	ProvinceCode         string    `json:"province_code"`
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	IsPrimary            bool      `json:"is_primary"`
}

// FindStreetMappings queries matching street-to-ward mappings for a given province and unaccented street name.
func (r *AddressRepository) FindStreetMappings(ctx context.Context, provinceCode, streetNameUnaccented string) ([]StreetWardMapping, error) {
	query := `
		SELECT m.id, m.street_name, m.street_name_unaccented, m.ward_code, COALESCE(w.name, m.ward_code),
		       m.province_code, m.latitude, m.longitude, m.is_primary
		FROM street_ward_mappings m
		LEFT JOIN administrative_units w ON m.ward_code = w.code
		WHERE (m.province_code = $1 OR $1 = '')
		  AND (m.street_name_unaccented = $2 OR $2 ILIKE '%' || m.street_name_unaccented || '%' OR m.street_name_unaccented % $2)
		ORDER BY m.is_primary DESC
		LIMIT 10
	`
	rows, err := r.pool.Query(ctx, query, provinceCode, streetNameUnaccented)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []StreetWardMapping
	for rows.Next() {
		var m StreetWardMapping
		if err := rows.Scan(
			&m.ID, &m.StreetName, &m.StreetNameUnaccented, &m.WardCode, &m.WardName,
			&m.ProvinceCode, &m.Latitude, &m.Longitude, &m.IsPrimary,
		); err == nil {
			mappings = append(mappings, m)
		}
	}
	return mappings, rows.Err()
}

// GetWardName fetches official name of a ward by its code.
func (r *AddressRepository) GetWardName(ctx context.Context, wardCode string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT name FROM administrative_units WHERE code = $1`, wardCode).Scan(&name)
	return name, err
}

func (r *AddressRepository) GetOwnedAddress(ctx context.Context, customerID, addressID uuid.UUID) (*domain.ShippingAddress, error) {
	if customerID == uuid.Nil {
		return nil, domain.ErrAddressNotFound
	}
	return r.getAddress(ctx, addressID, customerID)
}
func (r *AddressRepository) ValidateArea(ctx context.Context, wardCode, provinceCode string) error {
	var valid bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM administrative_units w JOIN administrative_units p ON w.parent_code=p.code WHERE w.code=$1 AND p.code=$2 AND w.level='WARD' AND p.level='PROVINCE')`, wardCode, provinceCode).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return &domain.ValidationError{Field: "ward_code", Message: "ward must belong to province"}
	}
	return nil
}
