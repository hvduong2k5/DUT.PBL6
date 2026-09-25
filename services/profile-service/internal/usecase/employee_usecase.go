package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/security"
	"github.com/omamx/profile-service/internal/repository"
)

type CreateEmployeeRequest struct {
	UserID               *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode         string     `json:"employee_code"`
	FullName             string     `json:"full_name"`
	PhoneNumber          string     `json:"phone_number"`
	IDCardNumber         string     `json:"id_card_number"` // Plaintext CCCD to be envelope encrypted
	DepartmentID         string     `json:"department_id"`
	Position             string     `json:"position"`
	ContractType         string     `json:"contract_type"`
	ContractStartDate    time.Time  `json:"contract_start_date"`
	ContractEndDate      *time.Time `json:"contract_end_date,omitempty"`
	FoodSafetyCertNo     string     `json:"food_safety_cert_no"`
	FoodSafetyCertExpiry *time.Time `json:"food_safety_cert_expiry,omitempty"`
}

type EmployeeDetailResponse struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode         string     `json:"employee_code"`
	FullName             string     `json:"full_name"`
	PhoneNumber          string     `json:"phone_number"`
	IDCardNumber         string     `json:"id_card_number"` // Decrypted via Vault KEK / In-memory DEK cache
	DepartmentID         string     `json:"department_id"`
	Position             string     `json:"position"`
	ContractType         string     `json:"contract_type"`
	ContractStartDate    time.Time  `json:"contract_start_date"`
	ContractEndDate      *time.Time `json:"contract_end_date,omitempty"`
	FoodSafetyCertNo     string     `json:"food_safety_cert_no"`
	FoodSafetyCertExpiry *time.Time `json:"food_safety_cert_expiry,omitempty"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type EmployeeUsecase struct {
	repo      *repository.EmployeeRepository
	encryptor *security.EnvelopeEncryptor
}

func NewEmployeeUsecase(repo *repository.EmployeeRepository, encryptor *security.EnvelopeEncryptor) *EmployeeUsecase {
	return &EmployeeUsecase{
		repo:      repo,
		encryptor: encryptor,
	}
}

// CreateEmployee creates a new employee profile with AES-256-GCM + Vault KEK Envelope Encryption.
func (u *EmployeeUsecase) CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*domain.EmployeeProfile, error) {
	// 1. Envelope Encrypt CCCD number
	payload, err := u.encryptor.Encrypt(ctx, req.IDCardNumber)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	emp := &domain.EmployeeProfile{
		ID:                   uuid.New(),
		UserID:               req.UserID,
		EmployeeCode:         req.EmployeeCode,
		FullName:             req.FullName,
		PhoneNumber:          req.PhoneNumber,
		IDCardEncrypted:      payload.Ciphertext,
		IDCardNonce:          payload.Nonce,
		EncryptedDEK:         payload.EncryptedDEK,
		KEKVersion:           payload.KEKVersion,
		DepartmentID:         req.DepartmentID,
		Position:             req.Position,
		ContractType:         req.ContractType,
		ContractStartDate:    req.ContractStartDate,
		ContractEndDate:      req.ContractEndDate,
		FoodSafetyCertNo:     req.FoodSafetyCertNo,
		FoodSafetyCertExpiry: req.FoodSafetyCertExpiry,
		Status:               "ACTIVE",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := u.repo.CreateEmployee(ctx, emp); err != nil {
		return nil, err
	}

	return emp, nil
}

// GetEmployeeDetail retrieves employee details and decrypts PII CCCD number.
func (u *EmployeeUsecase) GetEmployeeDetail(ctx context.Context, id uuid.UUID) (*EmployeeDetailResponse, error) {
	emp, err := u.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt CCCD using Vault Transit / short-lived in-memory DEK cache
	decryptedCCCD, err := u.encryptor.Decrypt(ctx, &security.EncryptedPayload{
		Ciphertext:   emp.IDCardEncrypted,
		Nonce:        emp.IDCardNonce,
		EncryptedDEK: emp.EncryptedDEK,
		KEKVersion:   emp.KEKVersion,
	})
	if err != nil {
		return nil, err
	}

	return &EmployeeDetailResponse{
		ID:                   emp.ID,
		UserID:               emp.UserID,
		EmployeeCode:         emp.EmployeeCode,
		FullName:             emp.FullName,
		PhoneNumber:          emp.PhoneNumber,
		IDCardNumber:         decryptedCCCD,
		DepartmentID:         emp.DepartmentID,
		Position:             emp.Position,
		ContractType:         emp.ContractType,
		ContractStartDate:    emp.ContractStartDate,
		ContractEndDate:      emp.ContractEndDate,
		FoodSafetyCertNo:     emp.FoodSafetyCertNo,
		FoodSafetyCertExpiry: emp.FoodSafetyCertExpiry,
		Status:               emp.Status,
		CreatedAt:            emp.CreatedAt,
		UpdatedAt:            emp.UpdatedAt,
	}, nil
}

// ListEmployees lists employee profiles.
func (u *EmployeeUsecase) ListEmployees(ctx context.Context, departmentID, status string, limit, offset int) ([]*domain.EmployeeProfile, error) {
	return u.repo.ListEmployees(ctx, departmentID, status, limit, offset)
}
