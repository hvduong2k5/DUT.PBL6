package domain

import (
	"time"

	"github.com/google/uuid"
)

// Department là Entity danh mục tổ chức xưởng kẹo
type Department struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ManagerID   *uuid.UUID `json:"manager_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EmployeeDEKUpdate phục vụ zero-downtime key rotation
type EmployeeDEKUpdate struct {
	EmployeeID   uuid.UUID
	EncryptedDEK []byte
	KEKVersion   int
}

// EmployeeProfile là Entity thuộc Sub-Domain Quản lý Nhân sự & Giám sát VSATTP OCOP
type EmployeeProfile struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode         string     `json:"employee_code"`
	FullName             string     `json:"full_name"`
	PhoneNumber          string     `json:"phone_number"`
	IDCardEncrypted      []byte     `json:"-"`
	IDCardNonce          []byte     `json:"-"`
	EncryptedDEK         []byte     `json:"-"`
	KEKVersion           int        `json:"kek_version"`
	DepartmentID         string     `json:"department_id"`
	Position             string     `json:"position"`
	ContractType         string     `json:"contract_type"` // FULLTIME, PARTTIME, SEASONAL
	ContractStartDate    time.Time  `json:"contract_start_date"`
	ContractEndDate      *time.Time `json:"contract_end_date,omitempty"`
	FoodSafetyCertNo     string     `json:"food_safety_cert_no"`
	FoodSafetyCertExpiry *time.Time `json:"food_safety_cert_expiry,omitempty"`
	Status               string     `json:"status"` // ACTIVE, ON_LEAVE, TERMINATED
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
