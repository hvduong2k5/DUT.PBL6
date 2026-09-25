package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
)

type EmployeeRepository struct {
	pool *pgxpool.Pool
}

func NewEmployeeRepository(pool *pgxpool.Pool) *EmployeeRepository {
	return &EmployeeRepository{pool: pool}
}

// CreateEmployee inserts a new employee profile with envelope encrypted PII fields.
func (r *EmployeeRepository) CreateEmployee(ctx context.Context, emp *domain.EmployeeProfile) error {
	query := `
		INSERT INTO employee_profiles (
			id, user_id, employee_code, full_name, phone_number,
			id_card_encrypted, id_card_nonce, encrypted_dek, kek_version,
			department_id, position, contract_type, contract_start_date, contract_end_date,
			food_safety_cert_no, food_safety_cert_expiry, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19
		)
	`
	_, err := r.pool.Exec(ctx, query,
		emp.ID, emp.UserID, emp.EmployeeCode, emp.FullName, emp.PhoneNumber,
		emp.IDCardEncrypted, emp.IDCardNonce, emp.EncryptedDEK, emp.KEKVersion,
		emp.DepartmentID, emp.Position, emp.ContractType, emp.ContractStartDate, emp.ContractEndDate,
		emp.FoodSafetyCertNo, emp.FoodSafetyCertExpiry, emp.Status, emp.CreatedAt, emp.UpdatedAt,
	)
	return err
}

// GetEmployeeByID retrieves employee profile by UUID.
func (r *EmployeeRepository) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*domain.EmployeeProfile, error) {
	query := `
		SELECT id, user_id, employee_code, full_name, phone_number,
		       id_card_encrypted, id_card_nonce, encrypted_dek, kek_version,
		       department_id, position, contract_type, contract_start_date, contract_end_date,
		       food_safety_cert_no, food_safety_cert_expiry, status, created_at, updated_at
		FROM employee_profiles
		WHERE id = $1
	`
	var emp domain.EmployeeProfile
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FullName, &emp.PhoneNumber,
		&emp.IDCardEncrypted, &emp.IDCardNonce, &emp.EncryptedDEK, &emp.KEKVersion,
		&emp.DepartmentID, &emp.Position, &emp.ContractType, &emp.ContractStartDate, &emp.ContractEndDate,
		&emp.FoodSafetyCertNo, &emp.FoodSafetyCertExpiry, &emp.Status, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee not found: %s", id)
		}
		return nil, err
	}
	return &emp, nil
}

// ListEmployees retrieves employees with optional filtering by department and status.
func (r *EmployeeRepository) ListEmployees(ctx context.Context, departmentID, status string, limit, offset int) ([]*domain.EmployeeProfile, error) {
	query := `
		SELECT id, user_id, employee_code, full_name, phone_number,
		       id_card_encrypted, id_card_nonce, encrypted_dek, kek_version,
		       department_id, position, contract_type, contract_start_date, contract_end_date,
		       food_safety_cert_no, food_safety_cert_expiry, status, created_at, updated_at
		FROM employee_profiles
		WHERE ($1 = '' OR department_id = $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, departmentID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*domain.EmployeeProfile
	for rows.Next() {
		var emp domain.EmployeeProfile
		if err := rows.Scan(
			&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FullName, &emp.PhoneNumber,
			&emp.IDCardEncrypted, &emp.IDCardNonce, &emp.EncryptedDEK, &emp.KEKVersion,
			&emp.DepartmentID, &emp.Position, &emp.ContractType, &emp.ContractStartDate, &emp.ContractEndDate,
			&emp.FoodSafetyCertNo, &emp.FoodSafetyCertExpiry, &emp.Status, &emp.CreatedAt, &emp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		employees = append(employees, &emp)
	}
	return employees, rows.Err()
}

// FindExpiringCertificates scans for active employees whose food safety certificates expire within thresholdDays.
func (r *EmployeeRepository) FindExpiringCertificates(ctx context.Context, thresholdDays int) ([]*domain.EmployeeProfile, error) {
	query := `
		SELECT id, user_id, employee_code, full_name, phone_number,
		       id_card_encrypted, id_card_nonce, encrypted_dek, kek_version,
		       department_id, position, contract_type, contract_start_date, contract_end_date,
		       food_safety_cert_no, food_safety_cert_expiry, status, created_at, updated_at
		FROM employee_profiles
		WHERE status = 'ACTIVE'
		  AND food_safety_cert_expiry IS NOT NULL
		  AND food_safety_cert_expiry <= CURRENT_DATE + ($1 * INTERVAL '1 day')
		  AND food_safety_cert_expiry >= CURRENT_DATE
		ORDER BY food_safety_cert_expiry ASC
	`
	rows, err := r.pool.Query(ctx, query, thresholdDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expiring []*domain.EmployeeProfile
	for rows.Next() {
		var emp domain.EmployeeProfile
		if err := rows.Scan(
			&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FullName, &emp.PhoneNumber,
			&emp.IDCardEncrypted, &emp.IDCardNonce, &emp.EncryptedDEK, &emp.KEKVersion,
			&emp.DepartmentID, &emp.Position, &emp.ContractType, &emp.ContractStartDate, &emp.ContractEndDate,
			&emp.FoodSafetyCertNo, &emp.FoodSafetyCertExpiry, &emp.Status, &emp.CreatedAt, &emp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		expiring = append(expiring, &emp)
	}
	return expiring, rows.Err()
}

// UpdateEmployeeEncryptedDEK updates rewrapped DEK and KEK version for key rotation job.
func (r *EmployeeRepository) UpdateEmployeeEncryptedDEK(ctx context.Context, id uuid.UUID, newEncryptedDEK []byte, newKEKVer int) error {
	query := `
		UPDATE employee_profiles
		SET encrypted_dek = $2, kek_version = $3, updated_at = NOW()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, id, newEncryptedDEK, newKEKVer)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("employee profile not found for key rotation")
	}
	return nil
}
