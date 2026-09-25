package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

// Create inserts a new customer profile.
func (r *CustomerRepository) Create(ctx context.Context, c *domain.CustomerProfile) error {
	prefBytes, err := json.Marshal(c.Preferences)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO customer_profiles (
			id, user_id, full_name, phone_number, email, date_of_birth, gender,
			avatar_url, preferences, status, version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`
	_, err = r.pool.Exec(ctx, query,
		c.ID, c.UserID, c.FullName, c.PhoneNumber, c.Email, c.DateOfBirth, c.Gender,
		c.AvatarURL, prefBytes, c.Status, c.Version, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetByID retrieves a customer profile by its primary UUID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, email, date_of_birth, gender,
		       avatar_url, preferences, status, version, created_at, updated_at
		FROM customer_profiles
		WHERE id = $1
	`
	var c domain.CustomerProfile
	var prefBytes []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.UserID, &c.FullName, &c.PhoneNumber, &c.Email, &c.DateOfBirth, &c.Gender,
		&c.AvatarURL, &prefBytes, &c.Status, &c.Version, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, err
	}

	if len(prefBytes) > 0 {
		var prefs map[string]any
		if err := json.Unmarshal(prefBytes, &prefs); err == nil {
			c.Preferences = prefs
		}
	}

	return &c, nil
}

// UpdateProfileWithOptimisticLock executes CAS update on customer profile.
// If RowsAffected == 0: executes a secondary check to distinguish 404 (Not Found) from 409 (Conflict).
func (r *CustomerRepository) UpdateProfileWithOptimisticLock(
	ctx context.Context,
	id uuid.UUID,
	fullName string,
	phone string,
	email string,
	expectedVersion int,
) (*domain.CustomerProfile, error) {
	query := `
		UPDATE customer_profiles
		SET full_name = $2,
		    phone_number = $3,
		    email = $4,
		    version = version + 1,
		    updated_at = NOW()
		WHERE id = $1 AND version = $5
		RETURNING id, user_id, full_name, phone_number, email, date_of_birth, gender,
		          avatar_url, preferences, status, version, created_at, updated_at
	`

	var updated domain.CustomerProfile
	var prefBytes []byte

	err := r.pool.QueryRow(ctx, query, id, fullName, phone, email, expectedVersion).Scan(
		&updated.ID, &updated.UserID, &updated.FullName, &updated.PhoneNumber, &updated.Email,
		&updated.DateOfBirth, &updated.Gender, &updated.AvatarURL, &prefBytes,
		&updated.Status, &updated.Version, &updated.CreatedAt, &updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Zero rows affected -> Secondary check to disambiguate 404 vs 409
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM customer_profiles WHERE id = $1)`
			checkErr := r.pool.QueryRow(ctx, checkQuery, id).Scan(&exists)
			if checkErr != nil {
				return nil, checkErr
			}

			if !exists {
				return nil, domain.ErrCustomerNotFound
			}
			return nil, domain.ErrOptimisticLockConflict
		}
		return nil, err
	}

	if len(prefBytes) > 0 {
		var prefs map[string]any
		if err := json.Unmarshal(prefBytes, &prefs); err == nil {
			updated.Preferences = prefs
		}
	}

	return &updated, nil
}
