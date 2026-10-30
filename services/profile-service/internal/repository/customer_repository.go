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
	if c.Gender == "" {
		c.Gender = "UNSPECIFIED"
	}
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

// CreateIfAbsent inserts a new customer profile unless one already exists for
// the same user_id. It backs identity provisioning: registration events may be
// delivered more than once, and the UNIQUE constraint on user_id makes the
// insert itself the idempotency guard. Returns true when a row was created.
func (r *CustomerRepository) CreateIfAbsent(ctx context.Context, c *domain.CustomerProfile) (bool, error) {
	if c.Gender == "" {
		c.Gender = "UNSPECIFIED"
	}
	prefBytes, err := json.Marshal(c.Preferences)
	if err != nil {
		return false, err
	}

	query := `
		INSERT INTO customer_profiles (
			id, user_id, full_name, phone_number, email, date_of_birth, gender,
			avatar_url, preferences, status, version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (user_id) DO NOTHING
	`
	tag, err := r.pool.Exec(ctx, query,
		c.ID, c.UserID, c.FullName, c.PhoneNumber, c.Email, c.DateOfBirth, c.Gender,
		c.AvatarURL, prefBytes, c.Status, c.Version, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SetStatusByUserID transitions the profile owned by userID to the given
// status. When onlyFrom is non-empty the transition applies only when the
// current status equals it, so a late lifecycle event cannot resurrect or
// demote a profile that has already moved on. Returns true when a row changed.
func (r *CustomerRepository) SetStatusByUserID(ctx context.Context, userID uuid.UUID, status domain.CustomerStatus, onlyFrom domain.CustomerStatus) (bool, error) {
	query := `
		UPDATE customer_profiles
		SET status = $2, updated_at = NOW()
		WHERE user_id = $1 AND status <> $2
	`
	args := []any{userID, string(status)}
	if onlyFrom != "" {
		query += ` AND status = $3`
		args = append(args, string(onlyFrom))
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// GetByID retrieves a customer profile by its primary UUID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, COALESCE(email, ''), date_of_birth, COALESCE(gender, 'UNSPECIFIED'),
		       COALESCE(avatar_url, ''), COALESCE(preferences, '{}'::jsonb), status, version, created_at, updated_at
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	query := `
        UPDATE customer_profiles
		SET full_name = $2,
		    phone_number = $3,
		    email = $4,
		    version = version + 1,
		    updated_at = NOW()
		WHERE id = $1 AND version = $5
		RETURNING id, user_id, full_name, phone_number, COALESCE(email, ''), date_of_birth, COALESCE(gender, 'UNSPECIFIED'),
		          COALESCE(avatar_url, ''), COALESCE(preferences, '{}'::jsonb), status, version, created_at, updated_at
	`

	var updated domain.CustomerProfile
	var prefBytes []byte

	err = tx.QueryRow(ctx, query, id, fullName, phone, email, expectedVersion).Scan(
		&updated.ID, &updated.UserID, &updated.FullName, &updated.PhoneNumber, &updated.Email,
		&updated.DateOfBirth, &updated.Gender, &updated.AvatarURL, &prefBytes,
		&updated.Status, &updated.Version, &updated.CreatedAt, &updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Zero rows affected -> Secondary check to disambiguate 404 vs 409
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM customer_profiles WHERE id = $1)`
			checkErr := tx.QueryRow(ctx, checkQuery, id).Scan(&exists)
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

	if err := insertEvent(ctx, tx, "CustomerProfile", id.String(), "vn.omama.profile.updated.v1", map[string]any{"customer_id": id, "new_version": updated.Version}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &updated, nil
}

// GetByUserID resolves the authentication subject to the separate profile ID.
func (r *CustomerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.CustomerProfile, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM customer_profiles WHERE user_id=$1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCustomerNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}
